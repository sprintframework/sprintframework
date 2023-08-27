/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package server

import (
	"github.com/sprintframework/sprint"
	"net/http"
	"google.golang.org/grpc"
)

type serverScanner struct {
	Scan         []interface{}
}

func ServerScanner(scan... interface{}) sprint.ServerScanner {
	return &serverScanner{
		Scan: scan,
	}
}

func (t *serverScanner) ServerBeans() []interface{} {
	beans := []interface{}{
		&struct {
			// make them visible
			Servers     []sprint.Server `inject:"optional"`
			GrpcServers []*grpc.Server `inject:"optional"`
			HttpServers []*http.Server `inject:"optional"`
		}{},
	}
	return append(beans, t.Scan...)
}

type grpcServerScanner struct {
	beanName    string
	Scan         []interface{}
}

func GrpcServerScanner(beanName string, scan... interface{}) sprint.ServerScanner {
	return &grpcServerScanner{
		beanName: beanName,
		Scan: scan,
	}
}

func (t *grpcServerScanner) ServerBeans() []interface{} {
	beans := []interface{}{
		AuthorizationMiddleware(),
		GrpcServerFactory(t.beanName),
		&struct {
			// make them visible
			Servers     []sprint.Server `inject:"optional"`
			GrpcServers []*grpc.Server `inject:"optional"`
			HttpServers []*http.Server `inject:"optional"`
		}{},
	}
	return append(beans, t.Scan...)
}

type httpServerScanner struct {
	beanName    string
	Scan         []interface{}
}

func HttpServerScanner(beanName string, scan... interface{}) sprint.ServerScanner {
	return &httpServerScanner{
		beanName: beanName,
		Scan: scan,
	}
}

func (t *httpServerScanner) ServerBeans() []interface{} {
	beans := []interface{}{
		HttpServerFactory(t.beanName),
		&struct {
			// make them visible
			HttpServers []*http.Server `inject`
		}{},
	}
	return append(beans, t.Scan...)
}


