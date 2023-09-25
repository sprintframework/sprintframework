/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintserver

import (
	"crypto/tls"
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	gracefulShutdownTimeout = 2 * time.Second
	shutdownTimeout = time.Second
)

type implGrpcServer struct {

	Properties         glue.Properties           `inject`
	Log                *zap.Logger               `inject`
	TlsConfig          *tls.Config               `inject:"optional"`

	NodeService        sprint.NodeService        `inject`

	beanName        string
	listenAddr      string

	srv             *grpc.Server
	listener        net.Listener

	alive           atomic.Bool
	shutdownOnce    sync.Once
	shutdownCh      chan struct{}
}

func NewGrpcServer(beanName string, srv *grpc.Server) sprint.Server {
	return &implGrpcServer{beanName: beanName, srv: srv, shutdownCh: make(chan struct{})}
}

func (t *implGrpcServer) PostConstruct() error {
	t.alive.Store(false)
	return nil
}

func (t *implGrpcServer) Bind() (err error) {

	t.listenAddr = t.Properties.GetString( fmt.Sprintf("%s.%s", t.beanName, "bind-address"), "")

	if t.listenAddr == "" {
		return errors.Errorf("property '%s.bind-address' not found in server context", t.beanName)
	}

	tcpAddr, err := sprintutils.ParseAndAdjustTCPAddr(t.listenAddr, t.NodeService.NodeSeq())
	if err != nil {
		return
	}
	t.listenAddr = fmt.Sprintf("%s:%d", tcpAddr.IP.String(), tcpAddr.Port)

	t.listener, err = net.Listen("tcp4", t.listenAddr)
	if err != nil {
		return err
	}

	if t.TlsConfig != nil {
		t.listener = tls.NewListener(t.listener, t.TlsConfig.Clone())
	}

	return nil
}

func (t *implGrpcServer) Alive() bool {
	return t.alive.Load()
}

func (t *implGrpcServer) ListenAddress() net.Addr {
	if t.listener != nil {
		return t.listener.Addr()
	} else {
		return sprint.EmptyAddr
	}
}

func (t *implGrpcServer) Shutdown() (err error) {

	t.shutdownOnce.Do(func() {

		t.Log.Info("GrpcServerShutdown",
			zap.String("addr", t.ListenAddress().String()),
			zap.String("network", t.ListenAddress().Network()))

		// notify everyone that we are shutting down
		close(t.shutdownCh)

		if !t.doGracefulStop() {
			t.doStop()
		}

		if t.listener != nil {
			t.listener.Close()
		}

	})

	return
}

func (t *implGrpcServer) doGracefulStop() bool {

	stopCh := make(chan struct{})
	go func() {
		t.srv.GracefulStop()
		close(stopCh)
	}()

	/**
	Wait a little bit for graceful shutdown of gRPC server
	*/

	select {
	case <-stopCh:
		return true
	case <-time.After(gracefulShutdownTimeout):
		return false
	}

	return true
}

func (t *implGrpcServer) doStop() bool {

	stopCh := make(chan struct{})
	go func() {
		t.srv.Stop()
		close(stopCh)
	}()

	select {
	case <-stopCh:
		return true
	case <-time.After(shutdownTimeout):
		return false
	}

	return true
}

func (t *implGrpcServer) ShutdownCh() <-chan struct{} {
	return t.shutdownCh
}

func (t *implGrpcServer) Destroy() error {
	// safe to call twice
	t.Shutdown()
	return nil
}

func (t *implGrpcServer) Serve() (err error) {

	defer sprintutils.PanicToError(&err)

	if t.TlsConfig != nil {
		t.Log.Info("GrpcServerServe",
			zap.String("addr", t.ListenAddress().String()),
			zap.String("network", t.ListenAddress().Network()),
			zap.Bool("tls", true),
			zap.Bool("insecure", t.TlsConfig.InsecureSkipVerify))
	} else {
		t.Log.Info("GrpcServerServe",
			zap.String("addr", t.ListenAddress().String()),
			zap.String("network", t.ListenAddress().Network()),
			zap.Bool("tls", false))
	}

	t.alive.Store(true)
	err = t.srv.Serve(t.listener)
	t.alive.Store(false)

	if err == nil || strings.Contains(err.Error(), "closed") {
		return nil
	}

	if err != nil {
		t.Log.Warn("GrpcServerClose", zap.Error(err))
	}
	return err

}

