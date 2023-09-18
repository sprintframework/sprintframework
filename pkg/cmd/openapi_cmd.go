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

type implOpenapiCommand struct {
	Application      sprint.Application      `inject`
	ResourceService sprint.ResourceService   `inject`
}

func OpenAPICommand() sprint.Command {
	return &implOpenapiCommand{}
}

func (t *implOpenapiCommand) BeanName() string {
	return "openapi"
}

func (t *implOpenapiCommand) Help() string {
	helpText := `
Usage: ./%s openapi

	Display Open API interfaces available in the application.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implOpenapiCommand) Synopsis() string {
	return "openapi description"
}

func (t *implOpenapiCommand) Run(args []string) error {
	print(t.ResourceService.GetOpenAPI("resources"))
	return nil
}
