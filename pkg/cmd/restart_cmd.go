/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
)

type implRestartCommand struct {
	Context glue.Context `inject`
}

func RestartCommand() sprint.Command {
	return &implRestartCommand{}
}

func (t *implRestartCommand) BeanName() string {
	return "restart"
}

func (t *implRestartCommand) Desc() string {
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
