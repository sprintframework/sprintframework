/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package server

import (
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"net/http"
	"google.golang.org/grpc"
)

type serverScanner struct {
	scan []interface{}
}

func ServerScanner(scan... interface{}) glue.Scanner {
	return &serverScanner{
		scan: scan,
	}
}

func (t *serverScanner) Beans() []interface{} {
	beans := []interface{}{
		&struct {
			// make them visible
			Servers     []sprint.Server `inject:"optional"`
			GrpcServers []*grpc.Server `inject:"optional"`
			HttpServers []*http.Server `inject:"optional"`
		}{},
	}
	return append(beans, t.scan...)
}

type grpcServerScanner struct {
	beanName string
	scan     []interface{}
}

func GrpcServerScanner(beanName string, scan... interface{}) glue.Scanner {
	return &grpcServerScanner{
		beanName: beanName,
		scan:     scan,
	}
}

func (t *grpcServerScanner) Beans() []interface{} {
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
	return append(beans, t.scan...)
}

type httpServerScanner struct {
	beanName string
	scan     []interface{}
}

func HttpServerScanner(beanName string, scan... interface{}) glue.Scanner {
	return &httpServerScanner{
		beanName: beanName,
		scan:     scan,
	}
}

func (t *httpServerScanner) Beans() []interface{} {
	beans := []interface{}{
		HttpServerFactory(t.beanName),
		&struct {
			// make them visible
			HttpServers []*http.Server `inject`
		}{},
	}
	return append(beans, t.scan...)
}


