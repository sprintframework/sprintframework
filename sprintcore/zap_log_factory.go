/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintcore

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
	"reflect"
)

type implZapLogFactory struct {
	Application      sprint.Application      `inject`
	ApplicationFlags sprint.ApplicationFlags `inject`
	Properties       glue.Properties         `inject`

	RotateLogger  *lumberjack.Logger       `inject:"optional"`

	LogDir         string        `value:"application.log.dir,default="`
	LogDirPerm     os.FileMode   `value:"application.perm.log.dir,default=-rwxrwxr-x"`
	LogFilePerm    os.FileMode   `value:"application.perm.log.file,default=-rw-rw-r--"`

}

func ZapLogFactory() glue.FactoryBean {
	return &implZapLogFactory{}
}

func (t *implZapLogFactory) Object() (object interface{}, err error) {

	defer sprintutils.PanicToError(&err)

	if t.ApplicationFlags.Daemon() {

		if t.RotateLogger != nil {

			writerSyncer := zapcore.AddSync(t.RotateLogger)

			encoderConfig := zap.NewProductionEncoderConfig()
			encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
			encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
			encoder := zapcore.NewConsoleEncoder(encoderConfig)

			core := zapcore.NewCore(encoder, writerSyncer, zapcore.DebugLevel)

			return zap.New(core, zap.AddCaller()), nil

		} else {

			logDir := t.LogDir
			if logDir == "" {
				logDir = filepath.Join(t.Application.ApplicationDir(), "log")
			}

			if err := sprintutils.CreateDirIfNeeded(logDir, t.LogDirPerm); err != nil {
				return nil, err
			}

			logDir = filepath.Join(logDir, t.getNodeName())

			if err := sprintutils.CreateDirIfNeeded(logDir, t.LogDirPerm); err != nil {
				return nil, err
			}

			logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", t.Application.Name()) )

			if err := sprintutils.CreateFileIfNeeded(logFile, t.LogFilePerm); err != nil {
				return nil, err
			}

			cfg := zap.NewDevelopmentConfig()
			cfg.OutputPaths = []string{
				logFile,
			}
			return cfg.Build()
		}

	} else {
		return zap.NewDevelopment()
	}

}

func (t *implZapLogFactory) ObjectType() reflect.Type {
	return sprint.ZapLogClass
}

func (t *implZapLogFactory) ObjectName() string {
	return "zap_logger"
}

func (t *implZapLogFactory) Singleton() bool {
	return true
}

func (t *implZapLogFactory) getNodeName() string {
	return sprintutils.AppendNodeSequence(t.Application.Name(), t.ApplicationFlags.Node())
}
