/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"strings"
)

type implResourcesCommand struct {
	Context           glue.Context               `inject`
	Application       sprint.Application         `inject`
	ResourceService   sprint.ResourceService     `inject`
}

func ResourcesCommand() sprint.Command {
	return &implResourcesCommand{}
}

func (t *implResourcesCommand) BeanName() string {
	return "resources"
}

func (t *implResourcesCommand) Help() string {
	helpText := `
Usage: ./%s resources [command]

	Provides management functionality over resources.

Commands:

  licenses                 Display the list of open source licenses using by the application.

  openapi                  Display Open API interfaces available in the application.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implResourcesCommand) Synopsis() string {
	return "resources commands: [licenses, openapi]"
}

func (t *implResourcesCommand) Run(args []string) error {
	if len(args) == 0 {
		return errors.Errorf("resources command needs argument, %s", t.Synopsis())
	}
	cmd := args[0]
	args = args[1:]

	switch cmd {
	case "licenses":
		content, err := t.ResourceService.GetLicenses("resources:licenses.txt")
		if err != nil {
			return err
		}
		print(content)
		return nil
	case "openapi":
		print(t.ResourceService.GetOpenAPI("resources"))
		return nil
	default:
		return errors.Errorf("unknown sub-command for resources '%s'", cmd)
	}

}
