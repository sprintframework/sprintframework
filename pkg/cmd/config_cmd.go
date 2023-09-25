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
	"github.com/sprintframework/sprintframework/pkg/app"
	"github.com/sprintframework/sprintframework/sprintutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"
)

type implConfigCommand struct {
	Context     glue.Context    `inject`
	Application sprint.Application `inject`
}

type coreConfigContext struct {
	ConfigRepository sprint.ConfigRepository `inject`
}

func ConfigCommand() sprint.Command {
	return &implConfigCommand{}
}

func (t *implConfigCommand) BeanName() string {
	return "config"
}

func (t *implConfigCommand) Help() string {
	helpText := `
Usage: ./%s config [command]

	Provides management functionality for the internal config.

Commands:

  get                      Gets the config entry by key.

  set                      Sets the config entry value by key.

  list                     List all config entries, hides the passwords and keys.

  dump                     Dumps all config entries to move to another system. Needs admin permissions.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implConfigCommand) Synopsis() string {
	return "config commands: [get, set, dump, list]"
}

func (t *implConfigCommand) Run(args []string) error {
	if len(args) == 0 {
		return errors.Errorf("config command needs argument, %s", t.Synopsis())
	}
	cmd := args[0]
	args = args[1:]
	switch cmd {
	case "get":
		return t.getConfig(args)

	case "set":
		return t.setConfig(args)

	case "dump", "list":
		return t.dumpConfig(cmd, args)

	default:
		return errors.Errorf("unknown sub-command for config '%s'", cmd)
	}

	return nil
}

func (t *implConfigCommand) getConfig(args []string) error {
	if len(args) < 1 {
		return errors.Errorf("'config get' command expected key argument: %v", args)
	}
	key := args[0]

	var value string
	err := sprint.DoWithControlClient(t.Context, func(client sprint.ControlClient) (err error) {
		value, err = client.ConfigCommand("get", []string {key})
		return
	})
	if err != nil && status.Code(err) == codes.Unavailable {
		value, err = t.getFromStorage(key)
	}
	if err != nil {
		return err
	}
	println(value)
	return nil
}

func (t *implConfigCommand) setConfig(args []string) error {
	if len(args) < 1 {
		return errors.Errorf("'config set' command expected key argument: %v", args)
	}

	key := args[0]
	args = args[1:]

	var value string
	if len(args) < 1 {
		if app.IsPEMProperty(key) {
			var err error
			value, err = sprintutils.PromptPEM("Enter PEM key: ")
			if err != nil {
				return err
			}
		} else if app.IsPasswordProperty(key) {
			value = sprintutils.PromptPassword("Enter password: ")
		} else {
			value = sprintutils.Prompt("Enter value: ")
		}

	} else {
		value = args[0]
		args = args[1:]
	}

	// value is the file path
	if strings.HasPrefix(value, "@") {
		filePath := value[1:]
		binVal, err := ioutil.ReadFile(filePath)
		if err != nil {
			return errors.Errorf("i/o error on reading value from file '%s', %v", filePath, err)
		}
		value = string(binVal)
	}

	err := sprint.DoWithControlClient(t.Context, func(client sprint.ControlClient) error {
		_, err := client.ConfigCommand("set", []string{ key, value })
		return err
	})

	if err != nil && status.Code(err) == codes.Unavailable  {
		fmt.Printf("Error on gRPC: %v\n", err)
		err = t.setInStorage(key, value)
	}
	if err != nil {
		return err
	}
	println("SUCCESS")
	return nil
}

func (t *implConfigCommand) dumpConfig(cmd string, args []string) error {
	err := sprint.DoWithControlClient(t.Context, func(client sprint.ControlClient) error {
		content, err := client.ConfigCommand(cmd, args)
		if err == nil {
			println(content)
		}
		return err
	})
	if err != nil && status.Code(err) == codes.Unavailable  {
		return t.dumpFromStorage(cmd, args, os.Stdout)
	}
	return err
}

func (t *implConfigCommand) dumpFromStorage(cmd string, args []string, writer io.StringWriter) (err error) {

	var prefix string
	if len(args) > 0 {
		prefix = args[0]
		args = args[1:]
	}

	limit := math.MaxInt64
	if cmd == "list" {
		limit = 80
	}

	if len(args) > 0 {
		limit, err = strconv.Atoi(args[0])
		if err != nil {
			return errors.Errorf("parsing limit '%s', %v", args[0], err)
		}
	}

	c := new(coreConfigContext)
	return doInCore(t.Context, c, func(core glue.Context) error {
		return c.ConfigRepository.EnumerateAll(prefix, func(key, value string) bool {
			if len(value) > limit {
				value = value[:limit] + "..."
				value = strings.ReplaceAll(value, "\n", " ")
			}
			writer.WriteString(fmt.Sprintf("%s: %s\n", key, value))
			return true
		})
	})
}

func (t *implConfigCommand) getFromStorage(key string) (value string, err error) {
	c := new(coreConfigContext)
	err = doInCore(t.Context, c, func(core glue.Context) error {
		value, err = c.ConfigRepository.Get(key)
		return err
	})
	return
}

func (t *implConfigCommand) setInStorage(key, value string) error {
	c := new(coreConfigContext)
	return doInCore(t.Context, c, func(core glue.Context) error {
		return c.ConfigRepository.Set(key, value)
	})
}
