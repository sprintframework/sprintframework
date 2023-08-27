/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"github.com/pkg/errors"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type implCertCommand struct {
	Context     glue.Context    `inject`
	Application sprint.Application `inject`
}

type coreDomainContext struct {
	CertificateService sprint.CertificateService `inject`
}

func CertCommand() sprint.Command {
	return &implCertCommand{}
}

func (t *implCertCommand) BeanName() string {
	return "cert"
}

func (t *implCertCommand) Desc() string {
	return "cert commands: [list, dump, upload, create, renew, remove, client, acme, self, manager]"
}

func (t *implCertCommand) Run(args []string) error {
	if len(args) == 0 {
		return errors.Errorf("cert command needs argument, %s", t.Desc())
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
		content, err :=  c.CertificateService.ExecuteCommand(cmd, args)
		if err != nil {
			return err
		}
		println(content)
		return nil
	})

}
