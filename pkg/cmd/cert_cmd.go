/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/cert"
	"github.com/sprintframework/sprint"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

type implCertCommand struct {
	Context     glue.Context    `inject`
	Application sprint.Application `inject`
}

type coreDomainContext struct {
	CertificateService cert.CertificateService `inject:"optional"`
}

func CertCommand() sprint.Command {
	return &implCertCommand{}
}

func (t *implCertCommand) BeanName() string {
	return "cert"
}

func (t *implCertCommand) Help() string {
	helpText := `
Usage: ./%s cert [command]

	Provides management functionality over certificates.

Commands:

  list                     Display list of all certificates in the system.

  dump                     Dumps the particular certificate to the console in PEM format.

  upload                   Uploads the certificate to the system.

  create                   Created the certificate.

  renew                    Renew certificate.

  remove                   Removed certificate from the system.

  client                   Client operations over certificates.

  acme                     ACME operations over certificates.

  self                     Self-signed certificates.

  manager                  Invoke certificate manager console.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implCertCommand) Synopsis() string {
	return "cert commands: [list, dump, upload, create, renew, remove, client, acme, self, manager]"
}

func (t *implCertCommand) Run(args []string) error {
	if len(args) == 0 {
		return errors.Errorf("cert command needs argument, %s", t.Synopsis())
	}
	cmd := args[0]
	args = args[1:]

	err := sprint.DoWithControlClient(t.Context, func(client sprint.ControlClient) error {
		content, err := client.CertificateCommand(cmd, args)
		if err == nil {
			println(content)
		}
		return err
	})
	if err == nil {
		return nil
	}
	if status.Code(err) != codes.Unavailable {
		return err
	}

	if cmd == "manager" {
		return errors.New("cert manager command available only on running server")
	}

	c := new(coreDomainContext)
	return doInCore(t.Context, c, func(core glue.Context) error {
		if c.CertificateService != nil {
			content, err :=  c.CertificateService.ExecuteCommand(cmd, args)
			if err != nil {
				return err
			}
			println(content)
		} else {
			println("Error: cert.CertificateService not found in core context")
		}
		return nil
	})

}
