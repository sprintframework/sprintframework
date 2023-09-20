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
	"github.com/sprintframework/cert"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/util"
	"reflect"
)

type implTlsConfigFactory struct {

	Properties     glue.Properties      `inject`
	NodeService    sprint.NodeService   `inject`

	CertificateManager cert.CertificateManager `inject:"optional"`

	beanName          string
}

func TlsConfigFactory(beanName string) glue.FactoryBean {
	return &implTlsConfigFactory{beanName: beanName}
}

func (t *implTlsConfigFactory) Object() (obj interface{}, err error) {

	defer util.PanicToError(&err)

	insecure := t.Properties.GetBool(fmt.Sprintf("%s.insecure", t.beanName), false)

	tlsConfig := &tls.Config{
		Rand:         rand.Reader,
		InsecureSkipVerify: insecure,
	}

	if t.CertificateManager != nil {
		tlsConfig.GetCertificate = t.CertificateManager.GetCertificate
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


