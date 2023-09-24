/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package client

import (
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"google.golang.org/grpc"
)

type clientScanner struct {
	scan []interface{}
}

func ClientScanner(scan... interface{}) glue.Scanner {
	return &clientScanner{
		scan: scan,
	}
}

func (t *clientScanner) Beans() []interface{} {
	beans := []interface{}{
		&struct {
			// make them visible
			ClientConn []*grpc.ClientConn `inject:"optional"`
			ControlClient []sprint.ControlClient `inject:"optional"`
		}{},
	}
	return append(beans, t.scan...)
}

type controlClientScanner struct {
	scan []interface{}
}

func ControlClientScanner(scan... interface{}) glue.Scanner {
	return &controlClientScanner{
		scan: scan,
	}
}

func (t *controlClientScanner) Beans() []interface{} {
	beans := []interface{}{
		GrpcClientFactory("control-grpc-client"),
		ControlClient(),
		&struct {
			// make them visible
			ClientConn []*grpc.ClientConn `inject:"optional"`
			ControlClient []sprint.ControlClient `inject:"optional"`
		}{},
	}
	return append(beans, t.scan...)
}

