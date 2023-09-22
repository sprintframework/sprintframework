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
	"sync"
)

type implHttpServer struct {

	Log             *zap.Logger            `inject`
	NodeService     sprint.NodeService     `inject`

	srv             *http.Server
	listener        net.Listener

	alive        atomic.Bool
	shutdownOnce sync.Once
	shutdownCh   chan struct{}
}

func NewHttpServer(srv *http.Server) sprint.Server {
	return &implHttpServer{srv: srv, shutdownCh: make(chan struct{})}
}

func (t *implHttpServer) PostConstruct() error {
	t.alive.Store(false)
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

func (t *implHttpServer) Alive() bool {
	return t.alive.Load()
}

func (t *implHttpServer) ListenAddress() net.Addr {
	if t.listener != nil {
		return t.listener.Addr()
	} else {
		return EmptyAddr{}
	}
}

func (t *implHttpServer) Shutdown() {

	t.Log.Info("HttpServerShutdown",
		zap.String("addr", t.ListenAddress().String()),
		zap.String("network", t.ListenAddress().Network()))

	t.shutdownOnce.Do(func() {
		if t.listener != nil {
			t.listener.Close()
		}
		t.srv.Close()
		close(t.shutdownCh)
	})
}

func (t *implHttpServer) ShutdownCh() <-chan struct{} {
	return t.shutdownCh
}

func (t *implHttpServer) Destroy() error {
	// safe to call twice
	t.Shutdown()
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

	if t.srv.TLSConfig != nil {
		t.listener = tls.NewListener(t.listener, t.srv.TLSConfig)
		//err = t.srv.ServeTLS(t.listener, "", "")
	}

	t.alive.Store(true)
	err = t.srv.Serve(t.listener)
	t.alive.Store(false)

	if err == nil || strings.Contains(err.Error(), "closed") {
		return nil
	}

	t.Log.Warn("HttpServerClose", zap.Error(err))
	return err
}
