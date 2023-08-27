/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package client

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"reflect"
	"github.com/pkg/errors"
)

type implAnyTlsConfigFactory struct {
	Properties    glue.Properties  `inject`
	beanName string
}

func AnyTlsConfigFactory(beanName string) glue.FactoryBean {
	return &implAnyTlsConfigFactory{beanName: beanName}
}

func (t *implAnyTlsConfigFactory) Object() (object interface{}, err error) {

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

	insecure := t.Properties.GetBool(fmt.Sprintf("%s.insecure", t.beanName), false)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecure,
		Rand:               rand.Reader,
	}

	tlsConfig.NextProtos = appendH2ToNextProtos(tlsConfig.NextProtos)
	return tlsConfig, nil
}

func (t *implAnyTlsConfigFactory) ObjectType() reflect.Type {
	return sprint.TlsConfigClass
}

func (t *implAnyTlsConfigFactory) ObjectName() string {
	return t.beanName
}

func (t *implAnyTlsConfigFactory) Singleton() bool {
	return true
}
