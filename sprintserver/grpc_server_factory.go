/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintserver

import (
	"fmt"
	"github.com/codeallergy/glue"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"reflect"
)

type implGrpcServerFactory struct {

	Properties              glue.Properties                `inject`
	Log                     *zap.Logger                    `inject`
	AuthorizationMiddleware sprint.AuthorizationMiddleware `inject`

	beanName  string
}

func GrpcServerFactory(beanName string) glue.FactoryBean {
	return &implGrpcServerFactory{beanName: beanName}
}

func (t *implGrpcServerFactory) Object() (object interface{}, err error) {

	defer sprintutils.PanicToError(&err)

	listenAddr := t.Properties.GetString( fmt.Sprintf("%s.%s", t.beanName, "bind-address"), "")

	t.Log.Info("GrpcServerFactory",
		zap.String("listenAddr", listenAddr),
		zap.String("bean", t.beanName))

	srv, err := t.createServer()
	if err != nil {
		return nil, err
	}

	return srv, nil
}

func (t *implGrpcServerFactory) ObjectType() reflect.Type {
	return sprint.GrpcServerClass
}

func (t *implGrpcServerFactory) ObjectName() string {
	return t.beanName
}

func (t *implGrpcServerFactory) Singleton() bool {
	return true
}

func (t *implGrpcServerFactory) createServer() (*grpc.Server, error) {

	var opts []grpc.ServerOption

	opts = append(opts, grpc.StreamInterceptor(grpc_auth.StreamServerInterceptor(t.AuthorizationMiddleware.Authenticate)))
	opts = append(opts, grpc.UnaryInterceptor(grpc_auth.UnaryServerInterceptor(t.AuthorizationMiddleware.Authenticate)))

	return grpc.NewServer(opts...), nil
}
