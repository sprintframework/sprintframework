/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"strings"
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

func (t *implHelpCommand) Help() string {
	helpText := `
Usage: ./%s help [command]

	Displays full text help for the command.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}


func (t *implHelpCommand) Synopsis() string {
	return "help command"
}

func (t *implHelpCommand) Run(args []string) error {

	if len(args) == 0 {
		fmt.Printf("Usage: ./%s help [command]\n", t.Application.Executable())

		for _, command := range t.Commands {
			fmt.Printf("    %s - %s\n", command.BeanName(), command.Synopsis())
		}
	} else {
		commandName := args[0]
		var found bool
		for _, command := range t.Commands {
			if command.BeanName() == commandName {
				fmt.Println(command.Help())
				found = true
				break
			}
		}
		if !found {
			return errors.Errorf("command '%s' not found", commandName)
		}
	}


	fmt.Println("Flags:")
	t.FlagSet.PrintDefaults()
	return nil
}
