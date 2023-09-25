/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintcmd

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"strings"
)

type implRestartNode struct {
	Application      sprint.Application      `inject`
	Context glue.Context `inject`
}

func RestartNode() *implRestartNode {
	return &implRestartNode{}
}

func (t *implRestartNode) BeanName() string {
	return "restart"
}

func (t *implRestartNode) Help() string {
	helpText := `
Usage: ./%s restart

	Restarts the application node.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implRestartNode) Synopsis() string {
	return "restart server"
}

func (t *implRestartNode) Run(args []string) error {

	return sprint.DoWithControlClient(t.Context, func(client sprint.ControlClient) error {
		status, err := client.Shutdown(true)
		if err == nil {
			println(status)
		}
		return err
	})

}
