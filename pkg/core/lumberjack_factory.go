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
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
	"reflect"
)

type implLumberjackFactory struct {
	Application      sprint.Application      `inject`
	ApplicationFlags sprint.ApplicationFlags `inject`
	Properties       glue.Properties      `inject`

	LogDir        string        `value:"application.log.dir,default="`
	LogDirPerm    os.FileMode   `value:"application.perm.log.dir,default=-rwxrwxr-x"`
	LogFilePerm   os.FileMode   `value:"application.perm.log.file,default=-rw-rw-r--"`

	MaxSize     int   `value:"lumberjack.max-size,default=500"`  // mb
	MaxBackups  int   `value:"lumberjack.max-backups,default=10"`
	MaxAge      int   `value:"lumberjack.max-age,default=28"` // days
	Compress    bool  `value:"lumberjack.compress,default=false"` // disabled by default
	Rotate      bool  `value:"lumberjack.rotate-on-start,default=false"` // disabled by default
}

func LumberjackFactory() glue.FactoryBean {
	return &implLumberjackFactory{}
}

func (t *implLumberjackFactory) Object() (object interface{}, err error) {

	logDir := t.LogDir
	if logDir == "" {
		logDir = filepath.Join(t.Application.ApplicationDir(), "log")
	}
	logFile := fmt.Sprintf("%s.log", t.Application.Name())

	if _, err := os.Stat(logDir); err != nil {
		if err = os.MkdirAll(logDir, t.LogDirPerm); err != nil {
			return nil, err
		}
	}

	logFile = filepath.Join(logDir, logFile)
	if err := util.CreateFileIfNeeded(logFile, t.LogFilePerm); err != nil {
		return nil, err
	}

	instance := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    t.MaxSize,
		MaxBackups: t.MaxBackups,
		MaxAge:     t.MaxAge,
		Compress:   t.Compress,
	}

	if t.Rotate {
		return instance, instance.Rotate()
	} else {
		return instance, nil
	}

}

func (t *implLumberjackFactory) ObjectType() reflect.Type {
	return sprint.LumberjackClass
}

func (t *implLumberjackFactory) ObjectName() string {
	return "lumberjack"
}

func (t *implLumberjackFactory) Singleton() bool {
	return true
}


