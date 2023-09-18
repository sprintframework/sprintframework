/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"fmt"
	"github.com/sprintframework/sprint"
	"strings"
)

type implLicensesCommand struct {
	Application      sprint.Application      `inject`
	ResourceService sprint.ResourceService   `inject`
}

func LicensesCommand() sprint.Command {
	return &implLicensesCommand{}
}

func (t *implLicensesCommand) BeanName() string {
	return "licenses"
}

func (t *implLicensesCommand) Help() string {
	helpText := `
Usage: ./%s licenses

	Display the list of open source licenses using by the application.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implLicensesCommand) Synopsis() string {
	return "open source licenses"
}

func (t *implLicensesCommand) Run(args []string) error {
	content, err := t.ResourceService.GetLicenses("resources:licenses.txt")
	if err != nil {
		return err
	}
	print(content)
	return nil
}
