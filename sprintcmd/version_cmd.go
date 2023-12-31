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

type implVersionCommand struct {
	Application sprint.Application `inject`
	Properties  glue.Properties `inject`

	Copyright      string   `value:"application.copyright,default="`
}

func VersionCommand() sprint.Command {
	return &implVersionCommand{}
}

func (t *implVersionCommand) BeanName() string {
	return "version"
}

func (t *implVersionCommand) Help() string {
	helpText := `
Usage: ./%s version

	Display the current version and build of the application.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implVersionCommand) Synopsis() string {
	return "show version"
}

func (t *implVersionCommand) Run(args []string) error {
	fmt.Printf("%s [Version %s, Build %s]\n", t.Application.Name(), t.Application.Version(), t.Application.Build())
	if t.Copyright != "" {
		fmt.Printf("%s\n", t.Copyright)
	}
	return nil
}
