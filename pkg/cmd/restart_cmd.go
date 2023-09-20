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

type implRestartCommand struct {
	Application      sprint.Application      `inject`
	Context glue.Context `inject`
}

func RestartCommand() sprint.Command {
	return &implRestartCommand{}
}

func (t *implRestartCommand) BeanName() string {
	return "restart"
}

func (t *implRestartCommand) Help() string {
	helpText := `
Usage: ./%s restart

	Restarts the application node.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implRestartCommand) Synopsis() string {
	return "restart server"
}

func (t *implRestartCommand) Run(args []string) error {

	return sprint.DoWithControlClient(t.Context, func(client sprint.ControlClient) error {
		status, err := client.Shutdown(true)
		if err == nil {
			println(status)
		}
		return err
	})

}
