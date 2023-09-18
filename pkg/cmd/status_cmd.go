/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"strings"
)

type implStatusCommand struct {
	Application sprint.Application `inject`
	Context glue.Context `inject`
}

func StatusCommand() sprint.Command {
	return &implStatusCommand{}
}

func (t *implStatusCommand) BeanName() string {
	return "status"
}

func (t *implStatusCommand) Help() string {
	helpText := `
Usage: ./%s status

	Returns the status of running server application.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implStatusCommand) Synopsis() string {
	return "server status"
}

func (t *implStatusCommand) Run(args []string) error {

	return sprint.DoWithControlClient(t.Context, func(client sprint.ControlClient) error {
		status, err := client.Status()
		if err == nil {
			println(status)
		}
		return err
	})

}
