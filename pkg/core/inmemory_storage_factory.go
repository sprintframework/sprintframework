/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"github.com/pkg/errors"
	"github.com/codeallergy/glue"
	"github.com/keyvalstore/cachestore"
	"reflect"
)

type implInmemoryStorageFactory struct {
	beanName        string
}

func InmemoryStorageFactory(beanName string) glue.FactoryBean {
	return &implInmemoryStorageFactory{beanName: beanName}
}

func (t *implInmemoryStorageFactory) Object() (object interface{}, err error) {

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

	return cachestore.New(t.beanName), nil
}

func (t *implInmemoryStorageFactory) ObjectType() reflect.Type {
	return cachestore.ObjectType()
}

func (t *implInmemoryStorageFactory) ObjectName() string {
	return t.beanName
}

func (t *implInmemoryStorageFactory) Singleton() bool {
	return true
}

