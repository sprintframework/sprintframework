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
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type implRunNode struct {
	Application                       sprint.Application                       `inject`
	ApplicationFlags                  sprint.ApplicationFlags                  `inject`
	SystemEnvironmentPropertyResolver sprint.SystemEnvironmentPropertyResolver `inject`
	Context                           glue.Context                             `inject`
	StartNode                         *implStartNode                           `inject`
	Children                          []glue.ChildContext                      `inject:"level=1"`

	LogDir         string        `value:"application.log.dir,default="`
	LogDirPerm     os.FileMode   `value:"application.perm.log.dir,default=-rwxrwxr-x"`
	LogFilePerm    os.FileMode   `value:"application.perm.log.file,default=-rw-rw-r--"`

	mutex     sync.Mutex
	startLog  *log.Logger
	logFile   *os.File
	logWriter io.Writer
}

func RunNode() *implRunNode {
	return &implRunNode{}
}

func (t *implRunNode) createLogFile() (string, error) {

	logDir := t.LogDir
	if logDir == "" {
		logDir = filepath.Join(t.Application.ApplicationDir(), "log")
	}

	if err := sprintutils.CreateDirIfNeeded(logDir, t.LogDirPerm); err != nil {
		return "", err
	}

	logDir = filepath.Join(logDir, t.getNodeName())

	if err := sprintutils.CreateDirIfNeeded(logDir, t.LogDirPerm); err != nil {
		return "", err
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("%s-start.log", t.Application.Name()) )
	return logFile, nil
}

func (t *implRunNode) lazyStartLog() *log.Logger {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	var err error
	if t.startLog == nil {

		t.logWriter = os.Stderr

		if t.ApplicationFlags.Daemon() {
			var fileName string
			fileName, err = t.createLogFile()
			if err != nil {
				fmt.Printf("Error: start log file name creation error, %v\n", err)
			} else {
				t.logFile, err = os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, t.LogFilePerm)
				if err != nil {
					fmt.Printf("Error: start log file '%s' creation error, %v\n", fileName, err)
				} else {
					t.logWriter = t.logFile
				}
			}
		}

		t.startLog = log.New(t.logWriter,
			"ERROR: ",
			log.Ldate|log.Ltime|log.Lshortfile)

	}
	return t.startLog
}

func (t *implRunNode) Destroy() error {
	if t.logFile != nil {
		return t.logFile.Close()
	}
	return nil
}

func (t *implRunNode) Run(args []string) (err error) {

	if t.ApplicationFlags.Verbose() {
		glue.Verbose(t.lazyStartLog())
	}

	var coreContext glue.ChildContext

	for _, child := range t.Children {
		if child.Role() == sprint.CoreRole {
			coreContext = child
			break
		}
	}

	if coreContext == nil {
		return errors.Errorf("core context not found in %+v", t.Children)
	}

	core, err := coreContext.Object()
	if err != nil {
		msg := fmt.Sprintf("core creation context failed by %v, used environment variables %+v", err, t.SystemEnvironmentPropertyResolver.Environ(false))
		t.lazyStartLog().Println(msg)
		return errors.New(msg)
	}

	logger, ok := findZapLogger(core)
	if !ok {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	defer func() {
		if r := recover(); r != nil {
			logger.Error("NodeRecover", zap.Error(err))
		}
	}()
	
	err = runServers(t.Application, t.ApplicationFlags, core, logger)
	if err != nil {
		logger.Error("NodeDestroyed",
			zap.Bool("restarting", t.Application.Restarting()),
			zap.Int("node", t.ApplicationFlags.Node()),
			zap.Strings("env", t.SystemEnvironmentPropertyResolver.Environ(false)),
			zap.Any("props", t.ApplicationFlags.Properties()),
			zap.Error(err))
	} else {
		logger.Info("NodeDestroyed",
			zap.Bool("restarting", t.Application.Restarting()),
			zap.Int("node", t.ApplicationFlags.Node()),
			zap.Strings("env", t.SystemEnvironmentPropertyResolver.Environ(false)),
			zap.Any("props", t.ApplicationFlags.Properties()))
	}

	err = core.Close()
	if err != nil {
		logger.Error("CoreContextClosed", zap.Error(err))
	} else {
		logger.Info("CoreContextClosed", )
	}

	logger.Sync()

	if t.Application.Restarting() {
		logger.Info("NodeRestarting")
		err = t.StartNode.Start(logger, true)
		if err != nil {
			logger.Error("NodeRestart",
				zap.Int("node", t.ApplicationFlags.Node()),
				zap.Strings("env", t.SystemEnvironmentPropertyResolver.Environ(false)),
				zap.Error(err))
		} else {
			logger.Info("NodeRestarted",
				zap.Int("node", t.ApplicationFlags.Node()),
				zap.Strings("env", t.SystemEnvironmentPropertyResolver.Environ(false)))
		}
	}

	return
}

func (t *implRunNode) getNodeName() string {
	return sprintutils.AppendNodeSequence(t.Application.Name(), t.ApplicationFlags.Node())
}

func findZapLogger(core glue.Context) (*zap.Logger, bool) {
	list := core.Bean(sprint.ZapLogClass, glue.DefaultLevel)
	if len(list) > 0 {
		if l, ok := list[0].Object().(*zap.Logger); ok {
			return l, true
		}
	}
	return nil, false
}