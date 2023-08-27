/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package server

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"reflect"
	"github.com/pkg/errors"
)

type implTlsConfigFactory struct {

	Properties  glue.Properties `inject`
	NodeService sprint.NodeService `inject`

	CertificateManager sprint.CertificateManager `inject`
	DomainService      sprint.CertificateService `inject`

	beanName          string
}

func TlsConfigFactory(beanName string) glue.FactoryBean {
	return &implTlsConfigFactory{beanName: beanName}
}

func (t *implTlsConfigFactory) Object() (obj interface{}, err error) {

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
		GetCertificate: t.CertificateManager.GetCertificate,
		Rand:         rand.Reader,
		InsecureSkipVerify: insecure,
	}

	tlsConfig.NextProtos = AppendH2ToNextProtos(tlsConfig.NextProtos)
	return tlsConfig, nil
}

func (t *implTlsConfigFactory) ObjectType() reflect.Type {
	return sprint.TlsConfigClass
}

func (t *implTlsConfigFactory) ObjectName() string {
	return t.beanName
}

func (t *implTlsConfigFactory) Singleton() bool {
	return true
}


