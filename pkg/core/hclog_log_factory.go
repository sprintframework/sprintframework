/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	"go.uber.org/zap"
	"reflect"
)

type implHCLogFactory struct {
	Log              *zap.Logger             `inject`
}

func HCLogFactory() glue.FactoryBean {
	return &implHCLogFactory{}
}

func (t *implHCLogFactory) Object() (object interface{}, err error) {

	defer sprintutils.PanicToError(&err)

	return newHCLogAdapter(t.Log), nil
}

func (t *implHCLogFactory) ObjectType() reflect.Type {
	return sprint.HCLogClass
}

func (t *implHCLogFactory) ObjectName() string {
	return "hclog_logger"
}

func (t *implHCLogFactory) Singleton() bool {
	return true
}

