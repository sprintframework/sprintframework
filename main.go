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
	"github.com/sprintframework/sprintframework/sprintapp"
	"github.com/sprintframework/sprintframework/sprintclient"
	"github.com/sprintframework/sprintframework/sprintcmd"
	"github.com/sprintframework/sprintframework/sprintcore"
	"github.com/sprintframework/sprintframework/sprintserver"
	"log"
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

	glue.Verbose(log.Default())

	beans := []interface{} {
		sprintapp.DefaultApplicationBeans,
		sprintapp.DefaultResources,
		sprintapp.DefaultAssets,
		sprintapp.DefaultGzipAssets,
		sprintcmd.DefaultCommands,

		glue.Child(sprint.CoreRole,
			sprintcore.DefaultCoreServices,
			sprintcore.BoltStoreFactory("config-store"),
			sprintcore.BadgerStoreFactory("secure-store"),
			sprintcore.AutoupdateService(),
			sprintcore.LumberjackFactory(),

			glue.Child(sprint.ServerRole,
				sprintserver.GrpcServerScanner("control-grpc-server"),
				sprintserver.ControlServer(),
				sprintserver.HttpServerFactory("control-gateway-server"),
				//sprintserver.TlsConfigFactory("tls-config"),
				sprintserver.TemplatePage("/", "resources:templates/index.tmpl"),
				),

			glue.Child(sprint.ServerRole,
				sprintserver.HttpServerScanner("redirect-https"),
				sprintserver.RedirectHttpsPage("redirect-https"),
				),
			),
		glue.Child(sprint.ControlClientRole,
			sprintclient.ControlClientBeans,
			//sprintclient.AnyTlsConfigFactory("client-tls-config"),
			),
	}

	return sprintapp.Application("sprint",
		sprintapp.WithVersion(Version),
		sprintapp.WithBuild(Build),
		sprintapp.WithBeans(beans)).
		Run(os.Args[1:])

}

func main() {

	if err := doMain(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
