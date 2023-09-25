/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintclient

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/codeallergy/properties"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	"path/filepath"
	"reflect"
)

var (
	CertFile = "client.crt"
	KeyFile  = "client.key"
)

type tlsConfigFactory struct {
	Application sprint.Application `inject`
	Properties  glue.Properties `inject`

	CompanyName   string        `value:"application.company,default=sprint"`

	beanName string
}

func TlsConfigFactory(beanName string) glue.FactoryBean {
	return &tlsConfigFactory{beanName: beanName}
}

func (t *tlsConfigFactory) Object() (object interface{}, err error) {

	defer sprintutils.PanicToError(&err)

	appDir := properties.Locate(t.CompanyName).GetDir(t.Application.Name())

	certFile := filepath.Join(appDir, CertFile)
	keyFile := filepath.Join(appDir, KeyFile)

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Errorf("LoadX509KeyPair for implControlClient SSL from %s and %s failed, %v", certFile, keyFile, err)
	}

	insecure := t.Properties.GetBool(fmt.Sprintf("%s.insecure", t.beanName), false)

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: insecure,
		Rand:               rand.Reader,
	}

	tlsConfig.NextProtos = appendH2ToNextProtos(tlsConfig.NextProtos)
	return tlsConfig, err
}

func (t *tlsConfigFactory) ObjectType() reflect.Type {
	return sprint.TlsConfigClass
}

func (t *tlsConfigFactory) ObjectName() string {
	return t.beanName
}

func (t *tlsConfigFactory) Singleton() bool {
	return true
}

