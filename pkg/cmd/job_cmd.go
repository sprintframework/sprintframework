/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"github.com/pkg/errors"
	"strings"
)

type implJobCommand struct {
	Application sprint.Application `inject`
	Context glue.Context `inject`
}

func JobCommand() sprint.Command {
	return &implJobCommand{}
}

func (t *implJobCommand) BeanName() string {
	return "job"
}

func (t *implJobCommand) Help() string {
	helpText := `
Usage: ./%s job [command]

	Provides management functionality for scheduled jobs.

Commands:

  list                      Gets the schedule list of all jobs.

  run                       Run a job by name.

  cancel                    Cancel the running job and remove from schedule list.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implJobCommand) Synopsis() string {
	return "job management - [list, run, cancel]"
}

func (t *implJobCommand) Run(args []string) error {

	if len(args) < 1 {
		return errors.Errorf("job command needs argument: %s",  t.Synopsis())
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