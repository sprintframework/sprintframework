/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type implStopCommand struct {
	Application      sprint.Application      `inject`
	ApplicationFlags sprint.ApplicationFlags `inject`
	Context          glue.Context         `inject`

	RunDir           string       `value:"application.run.dir,default="`
}

func StopCommand() sprint.Command {
	return &implStopCommand{}
}

func (t *implStopCommand) BeanName() string {
	return "stop"
}

func (t *implStopCommand) Help() string {
	helpText := `
Usage: ./%s stop

	Stops the running server application.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implStopCommand) Synopsis() string {
	return "stop server"
}

func (t *implStopCommand) Run(args []string) error {

	err := sprint.DoWithControlClient(t.Context, func(client sprint.ControlClient) error {
		status, err := client.Shutdown(false)
		if err == nil {
			println(status)
		}
		return err
	})

	if err != nil {
		return t.KillServer()
	}

	return nil
}

func (t *implStopCommand) KillServer() error {

	runDir := t.RunDir
	if runDir == "" {
		runDir = filepath.Join(t.Application.ApplicationDir(), "run")
	}
	pidFile := filepath.Join(runDir, fmt.Sprintf("%s.pid", t.Application.Name()))

	blob, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return err
	}

	pid := string(blob)

	if _, err := strconv.Atoi(pid); err != nil {
		return errors.Errorf("Invalid pid %s, %v", pid, err)
	}

	cmd := exec.Command("kill", "-2", pid)
	if err := cmd.Start(); err != nil {
		return err
	}

	if err := os.Remove(pidFile); err != nil {
		return errors.Errorf("Can not remove file %s, %v", pidFile, err)
	}

	return cmd.Wait()

}
