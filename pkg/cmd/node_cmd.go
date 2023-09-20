/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"strings"
)

type implNodeCommand struct {
	Context     glue.Context        `inject`
	Application sprint.Application  `inject`

	RunNode      *implRunNode         `inject`
	StartNode    *implStartNode       `inject`
	StopNode     *implStopNode        `inject`
	RestartNode  *implRestartNode     `inject`
	StatusNode   *implStatusNode      `inject`
}

func NodeCommand() sprint.Command {
	return &implNodeCommand{}
}

func (t *implNodeCommand) BeanName() string {
	return "node"
}

func (t *implNodeCommand) Help() string {
	helpText := `
Usage: ./%s node [command]

	Provides management functionality for the running node.

Commands:

  run                      Runs the application node as a foreground process.

  start                    Starts the application node in the background mode.

  stop                     Stops the already running node in the background mode.

  restart                  Restarts the application node in the background mode.

  status                   Returns the status of the running node application.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implNodeCommand) Synopsis() string {
	return "node commands: [run, start, stop, restart, status]"
}

func (t *implNodeCommand) Run(args []string) error {
	if len(args) == 0 {
		return errors.Errorf("node command needs argument, %s", t.Synopsis())
	}
	cmd := args[0]
	args = args[1:]
	switch cmd {
	case "run":
		return t.RunNode.Run(args)

	case "start":
		return t.StartNode.Run(args)

	case "stop":
		return t.StopNode.Run(args)

	case "restart":
		return t.RestartNode.Run(args)

	case "status":
		return t.StatusNode.Run(args)

	default:
		return errors.Errorf("unknown sub-command for config '%s'", cmd)
	}

	return nil
}
