/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package server

import (
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"reflect"
)

type implHealthcheckerFactory struct {
	glue.FactoryBean
	GrpcServer    *grpc.Server         `inject`

	enableServices  bool
}

func HealthcheckerFactory(enableServices bool) glue.FactoryBean {
	return &implHealthcheckerFactory{enableServices: enableServices}
}

func (t *implHealthcheckerFactory) Object() (object interface{}, err error) {

	defer sprintutils.PanicToError(&err)

	srv := health.NewServer()

	srv.SetServingStatus(
		"",
		grpc_health_v1.HealthCheckResponse_SERVING,
	)

	grpc_health_v1.RegisterHealthServer(t.GrpcServer, srv)

	if t.enableServices {
		for serviceName := range t.GrpcServer.GetServiceInfo() {
			srv.SetServingStatus(
				serviceName,
				grpc_health_v1.HealthCheckResponse_SERVING,
			)
		}
	}

	return srv, nil
}

func (t *implHealthcheckerFactory) ObjectType() reflect.Type {
	return sprint.HealthCheckerClass
}

func (t *implHealthcheckerFactory) ObjectName() string {
	return ""
}

func (t *implHealthcheckerFactory) Singleton() bool {
	return true
}
