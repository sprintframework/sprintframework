/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"flag"
	"fmt"
	"github.com/sprintframework/sprint"
)

type implHelpCommand struct {
	Application sprint.Application `inject`
	FlagSet     *flag.FlagSet     `inject`
	Commands    []sprint.Command   `inject:"lazy"`
}

func HelpCommand() sprint.Command {
	return &implHelpCommand{}
}

func (t *implHelpCommand) BeanName() string {
	return "help"
}

func (t *implHelpCommand) Desc() string {
	return "help command"
}

func (t *implHelpCommand) Run(args []string) error {

	fmt.Printf("Usage: ./%s [command]\n", t.Application.Executable())

	for _, command := range t.Commands {
		fmt.Printf("    %s - %s\n", command.BeanName(), command.Desc())
	}

	fmt.Println("Flags:")
	t.FlagSet.PrintDefaults()
	return nil
}
