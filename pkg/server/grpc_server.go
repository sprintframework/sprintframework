/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package server

import (
	"crypto/tls"
	"fmt"
	"github.com/pkg/errors"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/util"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"strings"
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

	running         atomic.Bool
}

func NewGrpcServer(beanName string, srv *grpc.Server) sprint.Server {
	return &implGrpcServer{beanName: beanName, srv: srv}
}

func (t *implGrpcServer) PostConstruct() error {
	t.running.Store(false)
	return nil
}

func (t *implGrpcServer) Bind() (err error) {

	t.listenAddr = t.Properties.GetString( fmt.Sprintf("%s.%s", t.beanName, "listen-address"), "")

	if t.listenAddr == "" {
		return errors.Errorf("property '%s.listen-address' not found in server context", t.beanName)
	}

	t.listenAddr, err = util.AdjustPortNumberInAddress(t.listenAddr, t.NodeService.NodeSeq())
	if err != nil {
		return err
	}

	t.listener, err = net.Listen("tcp4", t.listenAddr)
	if err != nil {
		return err
	}

	if t.TlsConfig != nil {
		t.listener = tls.NewListener(t.listener, t.TlsConfig.Clone())
	}

	return nil
}

func (t *implGrpcServer) Active() bool {
	return t.running.Load()
}

func (t *implGrpcServer) ListenAddress() net.Addr {
	if t.listener != nil {
		return t.listener.Addr()
	} else {
		return EmptyAddr{}
	}
}

func (t *implGrpcServer) Stop() {
	if t.running.CAS(true, false) {
		if t.listener != nil {
			t.listener.Close()
		}
		go t.srv.Stop()
	}
}

func (t *implGrpcServer) Destroy() error {
	t.Stop()
	return nil
}

func (t *implGrpcServer) Serve() (err error) {

	defer util.PanicToError(&err)

	t.Log.Info("GrpcServerServe",
		zap.String("addr", t.ListenAddress().String()),
		zap.String("network", t.ListenAddress().Network()),
		zap.Bool("tls", t.TlsConfig != nil))

	t.running.Store(true)
	err = t.srv.Serve(t.listener)
	t.running.Store(false)

	if err == nil || strings.Contains(err.Error(), "closed") {
		return nil
	}

	t.Log.Warn("GrpcServerClose", zap.Error(err))
	return err

}

