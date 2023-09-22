/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/keyvalstore/badgerstore"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/util"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

type implBadgerStorageFactory struct {
	beanName          string

	Log                               *zap.Logger                           `inject`
	Application                       sprint.Application                    `inject`
	ApplicationFlags                  sprint.ApplicationFlags               `inject`
	Properties                        glue.Properties                       `inject`
	SystemEnvironmentPropertyResolver sprint.SystemEnvironmentPropertyResolver `inject`

	DataDir           string       `value:"application.data.dir,default="`
	DataDirPerm       os.FileMode  `value:"application.perm.data.dir,default=-rwxrwx---"`

}

func BadgerStorageFactory(beanName string) glue.FactoryBean {
	return &implBadgerStorageFactory{beanName: beanName}
}

func (t *implBadgerStorageFactory) Object() (object interface{}, err error) {

	defer util.PanicToError(&err)

	bootstrapToken := t.Properties.GetString("application.boot", "")
	if bootstrapToken == "" {
		var ok bool
		bootstrapToken, ok = t.SystemEnvironmentPropertyResolver.PromptProperty("application.boot")
		if !ok || bootstrapToken == "" {
			return nil, errors.New("'application.boot' bootstrap token is required")
		}
	}

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

	dataDir = filepath.Join(dataDir, t.beanName)
	if err := util.CreateDirIfNeeded(dataDir, t.DataDirPerm); err != nil {
		return nil, err
	}

	splitKeyValueDirs := t.Properties.GetBool(fmt.Sprintf("%s.split-key-value", t.beanName), false)
	if splitKeyValueDirs {
		keyDataDir := filepath.Join(dataDir, "key")
		if err := util.CreateDirIfNeeded(keyDataDir, t.DataDirPerm); err != nil {
			return nil, err
		}
		valueDataDir := filepath.Join(dataDir, "value")
		if err := util.CreateDirIfNeeded(valueDataDir, t.DataDirPerm); err != nil {
			return nil, err
		}
	}

	storageKey, err := util.ParseToken(bootstrapToken)
	if err != nil {
		return nil, err
	}

	dataDirOpt := badgerstore.WithNope()
	if splitKeyValueDirs {
		dataDirOpt = badgerstore.WithKeyValueDir(dataDir)
	} else {
		dataDirOpt = badgerstore.WithDataDir(dataDir)
	}

	indexCacheSize := t.Properties.GetInt(fmt.Sprintf("%s.index-cache-size", t.beanName), 100 * 1024 * 1024)
	valueLogMaxEntries := t.Properties.GetInt(fmt.Sprintf("%s.value-log-max-entries", t.beanName), 1024 * 1024 * 1024)
	openTimeout := t.Properties.GetDuration(fmt.Sprintf("%s.open-timeout", t.beanName), time.Second)

	return badgerstore.New(t.beanName,
		dataDir,
		dataDirOpt,
		badgerstore.WithOpenTimeout(openTimeout),
		badgerstore.WithZapLogger(t.Log, t.Application.IsDev()),
		badgerstore.WithEncryptionKey(storageKey),
		badgerstore.WithIndexCacheSize(int64(indexCacheSize)),
		badgerstore.WithValueLogMaxEntries(uint32(valueLogMaxEntries)),
	)

}

func (t *implBadgerStorageFactory) ObjectType() reflect.Type {
	return badgerstore.ObjectType()
}

func (t *implBadgerStorageFactory) ObjectName() string {
	return t.beanName
}

func (t *implBadgerStorageFactory) Singleton() bool {
	return true
}

func (t *implBadgerStorageFactory) getNodeName() string {
	return util.AppendNodeSequence(t.Application.Name(), t.ApplicationFlags.Node())
}