/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"github.com/sprintframework/sprint"
)

type implOpenapiCommand struct {
	ResourceService sprint.ResourceService `inject`
}

func OpenAPICommand() sprint.Command {
	return &implOpenapiCommand{}
}

func (t *implOpenapiCommand) BeanName() string {
	return "openapi"
}

func (t *implOpenapiCommand) Desc() string {
	return "openapi description"
}

func (t *implOpenapiCommand) Run(args []string) error {
	print(t.ResourceService.GetOpenAPI("resources"))
	return nil
}
