/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package server

import (
	"crypto/tls"
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	rt "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"google.golang.org/protobuf/encoding/protojson"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type implHttpServerFactory struct {
	Log              *zap.Logger `inject`

	Properties       glue.Properties                   `inject`
	Pages            []sprint.Page                     `inject:"optional,level=1"`
	Resources        []*glue.ResourceSource            `inject:"optional"`
	AutocertManager  *autocert.Manager                 `inject:"optional"`
	TlsConfig        *tls.Config                       `inject:"optional"`

	beanName     string
}

func HttpServerFactory(beanName string) glue.FactoryBean {
	return &implHttpServerFactory{beanName: beanName}
}

func (t *implHttpServerFactory) isEnabled(name string) bool {
	return t.Properties.GetBool(fmt.Sprintf("%s.%s", t.beanName, name), false)
}

func (t *implHttpServerFactory) Object() (object interface{}, err error) {

	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = v
			case string:
				err = errors.New(v)
			default:
				err = errors.Errorf("%v", v)
			}
		}
	}()

	listenAddr := t.Properties.GetString(fmt.Sprintf("%s.%s", t.beanName, "listen-address"), "")

	if listenAddr == "" {
		return nil, errors.Errorf("property '%s.listen-address' not found in server context", t.beanName)
	}

	options := parseOptions(t.Properties.GetString(fmt.Sprintf("%s.%s", t.beanName, "options"), ""))

	mux := http.NewServeMux()

	if options["gateway"] {

		api := rt.NewServeMux(
			rt.WithMarshalerOption(runtime.MIMEWildcard, &rt.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					AllowPartial: t.isEnabled("allow-partial"),
					UseProtoNames: t.isEnabled("use-proto-names"),
					UseEnumNumbers: t.isEnabled("use-enum-numbers"),
					EmitUnpopulated: t.isEnabled("emit-unpopulated"),
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					AllowPartial: t.isEnabled("allow-partial"),
					DiscardUnknown: t.isEnabled("discard-unknown"),
				},
			}),
		)

		// reserve handler for API
		mux.Handle("/api/", api)
	}

	if t.AutocertManager != nil {
		mux.Handle("/.well-known/acme-challenge/", t.AutocertManager.HTTPHandler(nil))
	}

	visitedPatterns := make(map[string]bool)

	var pageList []string
	if options["pages"] {
		for _, page := range t.Pages {
			pattern := page.Pattern()
			if visitedPatterns[pattern] {
				t.Log.Warn("PatternExist", zap.String("pattern", pattern), zap.Any("page", page))
			} else {
				visitedPatterns[pattern] = true
				pageList = append(pageList, pattern)
				mux.Handle(pattern, page)
			}
		}
	}

	var assetList []string
	if options["assets"] {
		for pattern, handler := range t.groupAssets() {
			if visitedPatterns[pattern] {
				t.Log.Warn("PatternExist", zap.String("pattern", pattern))
			}
			visitedPatterns[pattern] = true
			assetList = append(assetList, pattern)
			mux.Handle(pattern, handler)
		}
	}

	readTimeout := t.Properties.GetDuration(fmt.Sprintf("%s.%s", t.beanName, "read-timeout"), 30 * time.Second)
	writeTimeout := t.Properties.GetDuration(fmt.Sprintf("%s.%s", t.beanName, "write-timeout"), 30 * time.Second)
	idleTimeout := t.Properties.GetDuration(fmt.Sprintf("%s.%s", t.beanName, "idle-timeout"), time.Minute)

	t.Log.Info("HTTPServerFactory",
		zap.String("listenAddr", listenAddr),
		zap.Strings("pages", pageList),
		zap.Strings("assets", assetList),
		zap.Any("options", options),
		zap.Bool("tls", t.TlsConfig != nil),
		zap.Bool("autocert", t.AutocertManager != nil))

	srv := &http.Server{
		Addr: listenAddr,
		Handler: mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout: idleTimeout,
	}

	if t.TlsConfig != nil {
		srv.TLSConfig = t.TlsConfig.Clone()
	}

	return srv, nil

}

func (t *implHttpServerFactory) ObjectType() reflect.Type {
	return sprint.HttpServerClass
}

func (t *implHttpServerFactory) ObjectName() string {
	return t.beanName
}

func (t *implHttpServerFactory) Singleton() bool {
	return true
}

type servingAsset struct {
	pattern string
	plainH  http.Handler
	gzipH   http.Handler
}

func (t *servingAsset) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if t.gzipH != nil && t.acceptGzip(r) {
		t.gzipH.ServeHTTP(w, r)
		return
	}
	if t.plainH != nil {
		t.plainH.ServeHTTP(w, r)
		return
	}
	http.Error(w, "resource not found", http.StatusNotFound)
}

func (t *servingAsset) acceptGzip(r *http.Request) bool {
	list := strings.Split(r.Header.Get(acceptEncoding), ",")
	for _, enc := range list {
		enc = strings.TrimSpace(enc)
		if "gzip" == enc {
			return true
		}
	}
	return false
}

func (t *implHttpServerFactory) groupAssets() map[string]*servingAsset {

	cache := make(map[string]*servingAsset)

	for _, res := range t.Resources {
		if strings.HasPrefix(res.Name, "assets") {

			var gzip bool
			var handler http.Handler
			handler = http.FileServer(res.AssetFiles)

			if strings.HasSuffix(res.Name, "gzip") {
				handler = gzipHeaderHandler { h: handler }
				gzip = true
			}

			for _, name := range res.AssetNames {

				Again:

				pattern := "/" + name
				s, ok := cache[pattern]
				if !ok {
					s = &servingAsset{pattern: pattern}
					cache[pattern] = s
				}

				if gzip {
					if s.gzipH != nil {
						t.Log.Warn("GzipHandlerExist", zap.String("pattern", pattern), zap.String("asset", name), zap.Any("files", res.AssetFiles))
					}
					s.gzipH = handler
				} else {
					if s.plainH != nil {
						t.Log.Warn("PlainHandlerExist", zap.String("pattern", pattern), zap.String("asset", name), zap.Any("files", res.AssetFiles))
					}
					s.plainH = handler
				}

				if name == "index.html" {
					name = ""
					goto Again
				}


			}


		}

	}

	return cache
}

type gzipHeaderHandler struct {
	h http.Handler
}

func (t gzipHeaderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.h.ServeHTTP(gzipWriter{w}, r)
}
