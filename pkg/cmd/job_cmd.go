/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"github.com/pkg/errors"
)

type implJobCommand struct {
	Context glue.Context `inject`
}

func JobCommand() sprint.Command {
	return &implJobCommand{}
}

func (t *implJobCommand) BeanName() string {
	return "job"
}

func (t *implJobCommand) Desc() string {
	return "job management - [list, run, cancel]"
}

func (t *implJobCommand) Run(args []string) error {

	if len(args) < 1 {
		return errors.New("job management commands: [list, run, cancel]")
	}

	command := args[0]
	args = args[1:]

	return sprint.DoWithControlClient(t.Context, func(client sprint.ControlClient) error {
		output, err := client.JobCommand(command, args)
		if err != nil {
			return err
		}
		println(output)
		return nil
	})

}