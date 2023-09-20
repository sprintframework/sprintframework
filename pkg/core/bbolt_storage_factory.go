/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/keyvalstore/bboltstore"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/util"
	"os"
	"path/filepath"
	"reflect"
)

type implBBoltStorageFactory struct {
	beanName        string

	Application      sprint.Application      `inject`
	ApplicationFlags sprint.ApplicationFlags `inject`
	Properties       glue.Properties         `inject`

	DataDir           string       `value:"application.data.dir,default="`
	DataDirPerm       os.FileMode  `value:"application.perm.data.dir,default=-rwxrwx---"`
	DataFilePerm      os.FileMode  `value:"application.perm.data.file,default=-rw-rw-r--"`
}

func BBoltStorageFactory(beanName string) glue.FactoryBean {
	return &implBBoltStorageFactory{beanName: beanName}
}

func (t *implBBoltStorageFactory) Object() (object interface{}, err error) {

	defer util.PanicToError(&err)

	dataDir := t.DataDir
	if dataDir == "" {
		dataDir = filepath.Join(t.Application.ApplicationDir(), "db")

		if err := util.CreateDirIfNeeded(dataDir, t.DataDirPerm); err != nil {
			return nil, err
		}

		dataDir = filepath.Join(dataDir, t.getNodeName())
	}
	if err := util.CreateDirIfNeeded(dataDir, t.DataDirPerm); err != nil {
		return nil, err
	}

	fileName := fmt.Sprintf("%s.db", t.beanName)
	dataFile := filepath.Join(dataDir, fileName)

	return bboltstore.New(t.beanName, dataFile, t.DataFilePerm)
}

func (t *implBBoltStorageFactory) ObjectType() reflect.Type {
	return bboltstore.ObjectType()
}

func (t *implBBoltStorageFactory) ObjectName() string {
	return t.beanName
}

func (t *implBBoltStorageFactory) Singleton() bool {
	return true
}

func (t *implBBoltStorageFactory) getNodeName() string {
	return util.FormatNodeName(t.Application.Name(), t.ApplicationFlags.Node())
}