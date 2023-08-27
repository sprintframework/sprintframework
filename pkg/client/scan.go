/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package client

import (
	"github.com/sprintframework/sprint"
	"google.golang.org/grpc"
)

type clientScanner struct {
	Name          string
	Scan         []interface{}
}

func ClientScanner(scannerName string, scan... interface{}) sprint.ClientScanner {
	return &clientScanner{
		Name: scannerName,
		Scan: scan,
	}
}

func (t *clientScanner) ScannerName() string {
	return t.Name
}

func (t *clientScanner) ClientBeans() []interface{} {
	beans := []interface{}{
		&struct {
			// make them visible
			ClientConn []*grpc.ClientConn `inject:"optional"`
			ControlClient []sprint.ControlClient `inject:"optional"`
		}{},
	}
	return append(beans, t.Scan...)
}

type controlClientScanner struct {
	Scan         []interface{}
}

func ControlClientScanner(scan... interface{}) sprint.ClientScanner {
	return &controlClientScanner{
		Scan: scan,
	}
}

func (t *controlClientScanner) ScannerName() string {
	return "control"
}

func (t *controlClientScanner) ClientBeans() []interface{} {
	beans := []interface{}{
		GrpcClientFactory("control-grpc-client"),
		ControlClient(),
		&struct {
			// make them visible
			ClientConn []*grpc.ClientConn `inject:"optional"`
			ControlClient []sprint.ControlClient `inject:"optional"`
		}{},
	}
	return append(beans, t.Scan...)
}

