/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintcmd

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

type implStartNode struct {
	Application                       sprint.Application                       `inject`
	ApplicationFlags                  sprint.ApplicationFlags                  `inject`
	Properties                        glue.Properties                       `inject`
	Registrars                        []sprint.FlagSetRegistrar                `inject`
	SystemEnvironmentPropertyResolver sprint.SystemEnvironmentPropertyResolver `inject`

	BootstrapTokens   []string    `value:"application.bootstrap-tokens,default="`
	Autoupdate         bool       `value:"application.autoupdate,default=false"`

	RunDir           string       `value:"application.run.dir,default="`
	RunDirPerm       os.FileMode  `value:"application.perm.run.dir,default=-rwxrwxr-x"`
	ExeFilePerm      os.FileMode  `value:"application.perm.exe.file,default=-rwxrwxr-x"`
	PidFilePerm      os.FileMode  `value:"application.perm.pid.file,default=-rw-rw-rw-"`
}

func StartNode() *implStartNode {
	return &implStartNode{}
}

func NewMinimalConfig() zap.Config {
	return zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Encoding:         "console",
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
}

func (t *implStartNode) Run(args []string) error {

	logger, err := NewMinimalConfig().Build()
	if err != nil {
		return err
	}

	/**
	Prompt all required tokens before start, so we can pass them through to child process environment
	 */
	for _, token := range t.BootstrapTokens {
		t.SystemEnvironmentPropertyResolver.PromptProperty(fmt.Sprintf("application.%s", token))
	}

	return t.Start(logger, false)
}

func (t *implStartNode) Start(logger *zap.Logger, restart bool) error {

	runDir := t.RunDir
	if runDir == "" {
		runDir = filepath.Join(t.Application.ApplicationDir(), "run")
	}
	pidFile := filepath.Join(runDir, fmt.Sprintf("%s.pid", t.getNodeName()))

	if err := sprintutils.CreateDirIfNeeded(runDir, t.RunDirPerm); err != nil {
		return err
	}

	_, err := os.Stat(pidFile)
	pidFileExist := err == nil

	if !restart && pidFileExist {
		pidContent, err := ioutil.ReadFile(pidFile)
		if err != nil {
			return errors.Errorf("io error on '%s', %v", pidFile, err)
		}
		pid, err := strconv.ParseInt(strings.TrimSpace(string(pidContent)), 10, 64)
		if err != nil {
			return errors.Errorf("invalid pid number in '%s', %v", pidFile, err)
		}
		process, err := os.FindProcess(int(pid))
		if err == nil && process.Signal(syscall.Signal(0)) == nil {
			return errors.Errorf("found already running process under pid '%d' from file '%s'", process.Pid, pidFile)
		}
	}

	args := []string{"-d"}

	for _, reg := range t.Registrars {
		args = reg.RegisterServerArgs(args)
	}

	executable, _ := os.Executable()
	executableDir := filepath.Dir(executable)
	fileName := filepath.Base(executable)

	var nextExePath string
	if t.Autoupdate {
		nextExePath = filepath.Join(executableDir, t.executableNext(fileName))
	} else {
		nextExePath = executable
	}

	logger.Info("NodeStart", zap.String("exePath", nextExePath), zap.String("username", User()), zap.Bool("autoupdate", t.Autoupdate))

	var updateOnStart bool
	var autoupdatePath string
	if t.Autoupdate {

		autoupdateNames := []string {
			t.Application.Name(),
			fmt.Sprintf("%s_%s", t.Application.Name(), runtime.GOOS),
			fmt.Sprintf("%s_%s_%s", t.Application.Name(), runtime.GOARCH, runtime.GOOS),
		}

		for _, appName := range autoupdateNames {
			autoupdatePath = filepath.Join(executableDir, appName)
			if _, err = os.Stat(autoupdatePath); err == nil {
				updateOnStart = true
				break
			}
			logger.Info("NoUpdatePath", zap.String("autoupdatePath", autoupdatePath), zap.Bool("updateOnStart", updateOnStart))
		}

	}

	if updateOnStart {
		logger.Info("UpdateOnStart", zap.String("autoupdatePath", autoupdatePath))
		if err := sprintutils.RemoveFileIfExist(nextExePath); err != nil {
			logger.Error("UpdateFile", zap.String("nextExePath", nextExePath), zap.Error(err))
		}
		if cnt, err := sprintutils.CopyFile(autoupdatePath, nextExePath, t.ExeFilePerm); err != nil {
			logger.Error("CopyFile", zap.String("from", autoupdatePath), zap.String("to", nextExePath), zap.Error(err))
			nextExePath = autoupdatePath
		} else {
			logger.Info("UpdateDone", zap.String("from", autoupdatePath), zap.String("to", nextExePath), zap.Int64("written", cnt))
			args = append(args, "-p", fmt.Sprintf("autoupdate.file=%s", autoupdatePath))
		}
	}

	args = append(args, "node", "run")
	cmd := exec.Command(nextExePath, args...)
	cmd.Env = append(os.Environ(), t.SystemEnvironmentPropertyResolver.Environ(true)...)

	if err := cmd.Start(); err != nil {
		logger.Error("Run",  zap.String("binary", nextExePath), zap.Strings("args", args), zap.Error(err))
		return err
	} else {
		logger.Info("Run", zap.String("binary", nextExePath), zap.Strings("args", args))
	}

	logger.Info("Daemon", zap.Int("pid", cmd.Process.Pid))

	content := fmt.Sprintf("%d", cmd.Process.Pid)

	err = ioutil.WriteFile(pidFile, []byte(content), 0666)
	if err != nil {
		logger.Error("WritePidFile", zap.String("pidFile", pidFile), zap.Error(err))
	} else if !pidFileExist {
		err = os.Chmod(pidFile, t.PidFilePerm)
		if err != nil {
			logger.Error("ChmodPidFile", zap.String("pidFile", pidFile), zap.Error(err))
		}
	}

	// detach child process
	err = cmd.Process.Release()
	if err != nil {
		logger.Error("ProcessRelease", zap.Error(err))
	}

	return err
}

func (t *implStartNode) executableNext(current string) string {

	name := t.getNodeName()
	odd := fmt.Sprintf(".%s.odd", name)
	even := fmt.Sprintf(".%s.even", name)

	if current == odd {
		return even
	} else {
		return odd
	}

}

func (t *implStartNode) getNodeName() string {
	return sprintutils.AppendNodeSequence(t.Application.Name(), t.ApplicationFlags.Node())
}

func User() string {
	user, err := user.Current()
	if err != nil {
		return ""
	}
	return user.Username
}

