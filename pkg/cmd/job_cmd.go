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

type implJobsCommand struct {
	Application sprint.Application `inject`
	Context glue.Context `inject`
}

func JobsCommand() sprint.Command {
	return &implJobsCommand{}
}

func (t *implJobsCommand) BeanName() string {
	return "jobs"
}

func (t *implJobsCommand) Help() string {
	helpText := `
Usage: ./%s jobs [command]

	Provides management functionality for scheduled jobs.

Commands:

  list                      Gets the schedule list of all jobs.

  run                       Run a job by name.

  cancel                    Cancel the running job and remove from schedule list.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implJobsCommand) Synopsis() string {
	return "jobs management - [list, run, cancel]"
}

func (t *implJobsCommand) Run(args []string) error {

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