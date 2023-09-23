/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package main

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprintframework/pkg/app"
	"github.com/sprintframework/sprintframework/pkg/client"
	"github.com/sprintframework/sprintframework/pkg/cmd"
	"github.com/sprintframework/sprintframework/pkg/core"
	"github.com/sprintframework/sprintframework/pkg/server"
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

	return app.Application("sprint",
		app.WithVersion(Version),
		app.WithBuild(Build),
		app.Beans(app.DefaultApplicationBeans, app.DefaultResources, app.DefaultAssets, app.DefaultGzipAssets, cmd.DefaultCommands),
		app.Core(core.CoreScanner(
			core.BoltStoreFactory("config-store"),
			core.BadgerStoreFactory("secure-store"),
			core.AutoupdateService(),
			core.LumberjackFactory(),
			)),
		app.Server(server.GrpcServerScanner("control-grpc-server",
			server.ControlServer(),
			server.HttpServerFactory("control-gateway-server"),
			//server.TlsConfigFactory("tls-config"),
			server.TemplatePage("/", "resources:templates/index.tmpl"),
			)),
		app.Server(server.HttpServerScanner("redirect-https", server.RedirectHttpsPage("redirect-https"))),
		app.Client(client.ClientScanner("control",
			client.GrpcClientFactory("control-grpc-client"),
			client.ControlClient(),
			//client.AnyTlsConfigFactory("client-tls-config"),
			)),
		).
		Run(os.Args[1:])

}

func main() {

	if err := doMain(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
