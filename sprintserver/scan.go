/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintserver

import (
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"google.golang.org/grpc"
	"net/http"
)

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
			GrpcServers []*grpc.Server  `inject:"optional"`
			HttpServers []*http.Server  `inject:"optional"`
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
			Servers     []sprint.Server `inject:"optional"`
			HttpServers []*http.Server `inject`
		}{},
	}
	return append(beans, t.scan...)
}

