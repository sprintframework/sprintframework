/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
)

type implStatusCommand struct {
	Context glue.Context `inject`
}

func StatusCommand() sprint.Command {
	return &implStatusCommand{}
}

func (t *implStatusCommand) BeanName() string {
	return "status"
}

func (t *implStatusCommand) Desc() string {
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
