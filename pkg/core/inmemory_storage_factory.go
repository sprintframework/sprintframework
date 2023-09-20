/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"github.com/codeallergy/glue"
	"github.com/keyvalstore/cachestore"
	"github.com/sprintframework/sprintframework/pkg/util"
	"reflect"
)

type implInmemoryStorageFactory struct {
	beanName        string
}

func InmemoryStorageFactory(beanName string) glue.FactoryBean {
	return &implInmemoryStorageFactory{beanName: beanName}
}

func (t *implInmemoryStorageFactory) Object() (object interface{}, err error) {

	defer util.PanicToError(&err)

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

