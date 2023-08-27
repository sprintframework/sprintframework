/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/pkg/errors"
	"os"
	"strings"
)

func indexStrings(a []string) map[string]bool {
	m := make(map[string]bool)
	for _, part := range a {
		key := strings.TrimSpace(part)
		m[key] = true
	}
	return m
}

func asList(m map[string]bool) []string {
	var a []string
	for k := range m {
		a = append(a, k)
	}
	return a
}

func createDirIfNeeded(dir string, perm os.FileMode) error {
	if _, err := os.Stat(dir); err != nil {
		if err = os.Mkdir(dir, perm); err != nil {
			return errors.Errorf("unable to create dir '%s' with permissions %x, %v", dir, perm ,err)
		}
		if err = os.Chmod(dir, perm); err != nil {
			return errors.Errorf("unable to chmod dir '%s' with permissions %x, %v", dir, perm ,err)
		}
	}
	return nil
}


func parseBool(str string) (bool, error) {
	switch str {
	case "1", "t", "T", "true", "TRUE", "True", "on", "ON", "On":
		return true, nil
	case "0", "f", "F", "false", "FALSE", "False", "off", "OFF", "Off":
		return false, nil
	}
	return false, errors.Errorf("invalid syntax %s", str)
}


func getKeyType(algorithm string) certcrypto.KeyType {
	switch strings.ToUpper(algorithm) {
	case "RSA2048":
		return certcrypto.RSA2048
	case "RSA4096":
		return certcrypto.RSA4096
	case "RSA8192":
		return certcrypto.RSA8192
	case "EC256":
		return certcrypto.EC256
	case "EC384":
		return certcrypto.EC384
	}
	return certcrypto.RSA2048
}
