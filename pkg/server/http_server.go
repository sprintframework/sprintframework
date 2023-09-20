/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package server

import (
	"crypto/tls"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/util"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"net"
	"net/http"
	"strings"
)

type implHttpServer struct {

	Log       *zap.Logger                 `inject`

	NodeService    sprint.NodeService     `inject`

	srv       *http.Server
	listener  net.Listener

	running   atomic.Bool
}

func NewHttpServer(srv *http.Server) sprint.Server {
	return &implHttpServer{srv: srv}
}

func (t *implHttpServer) PostConstruct() error {
	t.running.Store(false)
	return nil
}

func (t *implHttpServer) Bind() (err error) {

	t.srv.Addr, err = util.AdjustPortNumberInAddress(t.srv.Addr, t.NodeService.NodeSeq())
	if err != nil {
		return err
	}

	t.listener, err = net.Listen("tcp4", t.srv.Addr)
	if err != nil {
		return errors.Errorf("can not bind to port '%s', %v", t.srv.Addr, err)
	}

	return nil
}

func (t *implHttpServer) Active() bool {
	return t.running.Load()
}

func (t *implHttpServer) ListenAddress() net.Addr {
	if t.listener != nil {
		return t.listener.Addr()
	} else {
		return EmptyAddr{}
	}
}

func (t *implHttpServer) Stop() {
	if t.running.CAS(true, false) {
		if t.listener != nil {
			t.listener.Close()
		}
		t.srv.Close()
	}
}

func (t *implHttpServer) Destroy() error {
	t.Stop()
	return nil
}

func (t *implHttpServer) Serve() (err error) {

	defer util.PanicToError(&err)

	if t.srv.TLSConfig != nil {
		t.Log.Info("HttpServerServe",
			zap.String("addr", t.ListenAddress().String()),
			zap.String("network", t.ListenAddress().Network()),
			zap.Bool("tls", true),
			zap.Bool("insecure", t.srv.TLSConfig.InsecureSkipVerify))
	} else {
		t.Log.Info("HttpServerServe",
			zap.String("addr", t.ListenAddress().String()),
			zap.String("network", t.ListenAddress().Network()),
			zap.Bool("tls", false))
	}

	t.running.Store(true)
	if t.srv.TLSConfig != nil {
		t.listener = tls.NewListener(t.listener, t.srv.TLSConfig)
		//err = t.srv.ServeTLS(t.listener, "", "")
	}

	err = t.srv.Serve(t.listener)

	t.running.Store(false)

	if err == nil || strings.Contains(err.Error(), "closed") {
		return nil
	}

	t.Log.Warn("HttpServerClose", zap.Error(err))
	return err
}
