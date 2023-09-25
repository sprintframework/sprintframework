/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintcore

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/keyvalstore/bboltstore"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	"os"
	"path/filepath"
	"reflect"
)

type implBBoltStoreFactory struct {
	beanName        string

	Application      sprint.Application      `inject`
	ApplicationFlags sprint.ApplicationFlags `inject`
	Properties       glue.Properties         `inject`

	DataDir           string       `value:"application.data.dir,default="`
	DataDirPerm       os.FileMode  `value:"application.perm.data.dir,default=-rwxrwx---"`
	DataFilePerm      os.FileMode  `value:"application.perm.data.file,default=-rw-rw-r--"`
}

func BBoltStoreFactory(beanName string) glue.FactoryBean {
	return &implBBoltStoreFactory{beanName: beanName}
}

func (t *implBBoltStoreFactory) Object() (object interface{}, err error) {

	defer sprintutils.PanicToError(&err)

	dataDir := t.DataDir
	if dataDir == "" {
		dataDir = filepath.Join(t.Application.ApplicationDir(), "db")

		if err := sprintutils.CreateDirIfNeeded(dataDir, t.DataDirPerm); err != nil {
			return nil, err
		}

		dataDir = filepath.Join(dataDir, t.getNodeName())
	}
	if err := sprintutils.CreateDirIfNeeded(dataDir, t.DataDirPerm); err != nil {
		return nil, err
	}

	fileName := fmt.Sprintf("%s.db", t.beanName)
	dataFile := filepath.Join(dataDir, fileName)

	return bboltstore.New(t.beanName, dataFile, t.DataFilePerm)
}

func (t *implBBoltStoreFactory) ObjectType() reflect.Type {
	return bboltstore.ObjectType()
}

func (t *implBBoltStoreFactory) ObjectName() string {
	return t.beanName
}

func (t *implBBoltStoreFactory) Singleton() bool {
	return true
}

func (t *implBBoltStoreFactory) getNodeName() string {
	return sprintutils.AppendNodeSequence(t.Application.Name(), t.ApplicationFlags.Node())
}