/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintcore

import (
	"github.com/codeallergy/glue"
	"github.com/keyvalstore/cachestore"
	"github.com/sprintframework/sprintframework/sprintutils"
	"reflect"
)

type implCacheStoreFactory struct {
	beanName        string
}

func CacheStoreFactory(beanName string) glue.FactoryBean {
	return &implCacheStoreFactory{beanName: beanName}
}

func (t *implCacheStoreFactory) Object() (object interface{}, err error) {

	defer sprintutils.PanicToError(&err)

	return cachestore.New(t.beanName), nil
}

func (t *implCacheStoreFactory) ObjectType() reflect.Type {
	return cachestore.ObjectType()
}

func (t *implCacheStoreFactory) ObjectName() string {
	return t.beanName
}

func (t *implCacheStoreFactory) Singleton() bool {
	return true
}

