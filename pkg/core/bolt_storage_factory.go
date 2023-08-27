/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/keyvalstore/boltstore"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"os"
	"path/filepath"
	"reflect"
)

type implBoltStorageFactory struct {
	beanName        string

	Application sprint.Application `inject`
	Properties  glue.Properties `inject`

	DataDir           string       `value:"application.data.dir,default="`
	DataDirPerm       os.FileMode  `value:"application.perm.data.dir,default=-rwxrwx---"`
	DataFilePerm      os.FileMode  `value:"application.perm.data.file,default=-rw-rw-r--"`
}

func BoltStorageFactory(beanName string) glue.FactoryBean {
	return &implBoltStorageFactory{beanName: beanName}
}

func (t *implBoltStorageFactory) Object() (object interface{}, err error) {

	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = v
			case string:
				err = errors.New(v)
			default:
				err = errors.Errorf("%v", v)
			}
		}
	}()

	dataDir := t.DataDir
	if dataDir == "" {
		dataDir = filepath.Join(t.Application.ApplicationDir(), "db")

		if err := createDirIfNeeded(dataDir, t.DataDirPerm); err != nil {
			return nil, err
		}

		dataDir = filepath.Join(dataDir, t.Application.Name())
	}
	if err := createDirIfNeeded(dataDir, t.DataDirPerm); err != nil {
		return nil, err
	}

	fileName := fmt.Sprintf("%s.db", t.beanName)
	dataFile := filepath.Join(dataDir, fileName)

	return boltstore.New(t.beanName, dataFile, t.DataFilePerm)
}

func (t *implBoltStorageFactory) ObjectType() reflect.Type {
	return boltstore.ObjectType()
}

func (t *implBoltStorageFactory) ObjectName() string {
	return t.beanName
}

func (t *implBoltStorageFactory) Singleton() bool {
	return true
}


