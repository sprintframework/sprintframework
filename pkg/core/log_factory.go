/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/util"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
	"reflect"
)

type implLogFactory struct {
	Application      sprint.Application      `inject`
	ApplicationFlags sprint.ApplicationFlags `inject`
	Properties       glue.Properties         `inject`

	RotateLogger  *lumberjack.Logger       `inject:"optional"`

	LogDir         string        `value:"application.log.dir,default="`
	LogDirPerm     os.FileMode   `value:"application.perm.log.dir,default=-rwxrwxr-x"`
	LogFilePerm    os.FileMode   `value:"application.perm.log.file,default=-rw-rw-r--"`

}

func LogFactory() glue.FactoryBean {
	return &implLogFactory{}
}

func (t *implLogFactory) Object() (object interface{}, err error) {

	defer util.PanicToError(&err)

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

			if err := util.CreateDirIfNeeded(logDir, t.LogDirPerm); err != nil {
				return nil, err
			}

			logDir = filepath.Join(logDir, t.getNodeName())

			if err := util.CreateDirIfNeeded(logDir, t.LogDirPerm); err != nil {
				return nil, err
			}

			logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", t.Application.Name()) )

			if err := util.CreateFileIfNeeded(logFile, t.LogFilePerm); err != nil {
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

func (t *implLogFactory) ObjectType() reflect.Type {
	return sprint.LogClass
}

func (t *implLogFactory) ObjectName() string {
	return "zap_logger"
}

func (t *implLogFactory) Singleton() bool {
	return true
}

func (t *implLogFactory) getNodeName() string {
	return util.FormatNodeName(t.Application.Name(), t.ApplicationFlags.Node())
}
