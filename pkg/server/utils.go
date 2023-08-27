/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func parseOptions(str string) map[string]bool {
	cache := make(map[string]bool)
	parts := strings.Split(str, ";")
	for _, part := range parts {
		key := strings.TrimSpace(part)
		if len(key) > 0 {
			cache[key] = true
		}
	}
	return cache
}

type EmptyAddr struct {
}

func (t EmptyAddr) Network() string {
	return ""
}

func (t EmptyAddr) String() string {
	return ""
}

const alpnProtoStrH2 = "h2"

func AppendH2ToNextProtos(ps []string) []string {
	for _, p := range ps {
		if p == alpnProtoStrH2 {
			return ps
		}
	}
	ret := make([]string, 0, len(ps)+1)
	ret = append(ret, ps...)
	return append(ret, alpnProtoStrH2)
}


type rewriteHandler struct {
	from string
	to string
	delegate http.Handler
}

func (t *rewriteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if p == t.from {
		url := *r.URL
		url.Path = t.to
		r.URL = &url
	}
	t.delegate.ServeHTTP(w, r)
}

func (t *rewriteHandler) rewrite(p string) string {
	return p
}

func Rewrite(from, to string, handler http.Handler) http.Handler {
	return &rewriteHandler{
		from:  from,
		to:     to,
		delegate: handler,
	}
}

const (
	acceptEncoding  = "Accept-Encoding"
	acceptRanges    = "Accept-Ranges"
	contentEncoding = "Content-Encoding"
	contentLength   = "Content-Length"
)

type gzipHandler struct {
	handler http.Handler
}

type gzipWriter struct {
	w http.ResponseWriter
}

func (t gzipWriter) Header() http.Header {
	return t.w.Header()
}

func (t gzipWriter) Write(b []byte) (int, error) {
	return t.w.Write(b)
}

func (t gzipWriter) WriteHeader(statusCode int) {
	if statusCode == 200 {
		t.w.Header().Del(contentEncoding)
		t.w.Header().Set(contentEncoding, "gzip")
	}
	t.w.WriteHeader(statusCode)
}

type bufWriter struct {
	w http.ResponseWriter
	buf bytes.Buffer
	code int
}

func (t *bufWriter) Header() http.Header {
	return t.w.Header()
}

func (t *bufWriter) Write(b []byte) (int, error) {
	fmt.Printf("write len %d\n", len(b))
	n, err := t.buf.Write(b)
	fmt.Printf("written len %d\n", n)
	return n, err
}

func (t *bufWriter) Unpack() []byte {
	fmt.Printf("compressed len %d\n", len(t.buf.Bytes()))
	var plain bytes.Buffer
	zr, err := gzip.NewReader(&t.buf)
	if err != nil {
		// probably not gzip
		return t.buf.Bytes()
	}
	io.Copy(&plain, zr)
	zr.Close()
	return plain.Bytes()
}

func (t *bufWriter) WriteHeader(statusCode int) {
	fmt.Printf("write header %d\n", statusCode)
	t.code = statusCode
}

func (t gzipHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if t.acceptGzip(r) {
		fmt.Printf("Serve passthrought gzip %s, header %s\n", r.RequestURI, r.Header.Get(acceptEncoding))
		t.handler.ServeHTTP(gzipWriter{w}, r)
	} else {
		fmt.Printf("Serve decompress gzip %s, header %s\n", r.RequestURI, r.Header.Get(acceptEncoding))
		bw := bufWriter {
			w: w,
		}
		t.handler.ServeHTTP(&bw, r)

		if r.Method != "HEAD" {
			fmt.Printf("header content-length '%s'\n", w.Header().Get(contentLength))
			fmt.Printf("header content-range '%s'\n", w.Header().Get("Content-Range"))
			w.Header().Del(contentLength)
			w.Header().Del(acceptRanges)
			plain := bw.Unpack()
			fmt.Printf("plain size = '%d'\n", len(plain))
			fmt.Printf("plain = '%s'\n", string(plain))

			w.Header().Set(contentLength, strconv.Itoa(len(plain)))
			w.WriteHeader(bw.code)
			io.Copy(w, bytes.NewReader(plain))
		} else {
			w.WriteHeader(bw.code)
		}

	}

}

func (t gzipHandler) acceptGzip(r *http.Request) bool {
	list := strings.Split(r.Header.Get(acceptEncoding), ",")
	for _, enc := range list {
		enc = strings.TrimSpace(enc)
		if "gzip" == enc {
			return true
		}
	}
	return false
}

func GzipHandler(handler http.Handler) http.Handler {
	return gzipHandler {handler}
}