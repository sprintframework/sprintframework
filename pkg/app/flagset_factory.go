/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package app

import (
	"flag"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"reflect"
)

type implFlagSetFactory struct {
	Registrars []sprint.FlagSetRegistrar `inject`
}

func FlagSetFactory() glue.FactoryBean {
	return &implFlagSetFactory{}
}

func (t *implFlagSetFactory) Object() (interface{}, error) {
	fs := flag.NewFlagSet("sprint", flag.ContinueOnError)
	for _, reg := range t.Registrars {
		reg.RegisterFlags(fs)
	}
	return fs, nil
}

func (t *implFlagSetFactory) ObjectType() reflect.Type {
	return sprint.FlagSetClass
}

func (t *implFlagSetFactory) ObjectName() string {
	return ""
}

func (t *implFlagSetFactory) Singleton() bool {
	return true
}
