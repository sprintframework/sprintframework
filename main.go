/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package main

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/app"
	"github.com/sprintframework/sprintframework/pkg/client"
	"github.com/sprintframework/sprintframework/pkg/cmd"
	"github.com/sprintframework/sprintframework/pkg/core"
	"github.com/sprintframework/sprintframework/sprintserver"
	"os"
)

var (
	Version string
	Build   string
)

func doMain() (err error) {

	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = v
			case string:
				err = errors.New(v)
			default:
				err = errors.Errorf("%v", v)
			}
		}
	}()

	beans := []interface{} {
		app.ApplicationScanner(app.DefaultResources, app.DefaultAssets, app.DefaultGzipAssets, cmd.DefaultCommands),

		glue.Child(sprint.CoreRole,
			core.CoreScanner(),
			core.BoltStoreFactory("config-store"),
			core.BadgerStoreFactory("secure-store"),
			core.AutoupdateService(),
			core.LumberjackFactory(),

			glue.Child(sprint.ServerRole,
				sprintserver.GrpcServerScanner("control-grpc-server"),
				sprintserver.ControlServer(),
				sprintserver.HttpServerFactory("control-gateway-server"),
				//server.TlsConfigFactory("tls-config"),
				sprintserver.TemplatePage("/", "resources:templates/index.tmpl"),
				),

			glue.Child(sprint.ServerRole,
				sprintserver.HttpServerScanner("redirect-https"),
				sprintserver.RedirectHttpsPage("redirect-https"),
				),
			),
		glue.Child(sprint.ControlClientRole,
			client.ClientScanner(),
			client.GrpcClientFactory("control-grpc-client"),
			client.ControlClient(),
			//client.AnyTlsConfigFactory("client-tls-config"),
			),
	}

	return app.Application("sprint",
		app.WithVersion(Version),
		app.WithBuild(Build),
		app.WithBeans(beans)).
		Run(os.Args[1:])

}

func main() {

	if err := doMain(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
