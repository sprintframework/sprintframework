/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"github.com/sprintframework/sprint"
)

type implLicensesCommand struct {
	ResourceService sprint.ResourceService `inject`
}

func LicensesCommand() sprint.Command {
	return &implLicensesCommand{}
}

func (t *implLicensesCommand) BeanName() string {
	return "licenses"
}

func (t *implLicensesCommand) Desc() string {
	return "show all licenses"
}

func (t *implLicensesCommand) Run(args []string) error {
	content, err := t.ResourceService.GetLicenses("resources:licenses.txt")
	if err != nil {
		return err
	}
	print(content)
	return nil
}
