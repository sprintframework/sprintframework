/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintapp

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprintframework/sprintutils"
	"os"
	"strings"
	"sync"
)

type systemEnvironmentPropertyResolver struct {
	applicationName string
	priority int

	sync.Mutex
	cache map[string]string
}

func SystemEnvironmentPropertyResolver(applicationName string, priority int) glue.PropertyResolver {
	return &systemEnvironmentPropertyResolver{
		applicationName: applicationName,
		priority: priority,
		cache: make(map[string]string),
	}
}

func (t *systemEnvironmentPropertyResolver) String() string {
	return fmt.Sprintf("SystemEnvironmentPropertyResolver{%s,%d}", t.applicationName, t.priority)
}

func (t *systemEnvironmentPropertyResolver) Priority() int {
	return t.priority
}

func (t *systemEnvironmentPropertyResolver) GetProperty(key string) (string, bool) {
	if env, ok := t.toEnv(key); ok {

		t.Lock()
		value, ok := t.cache[env]
		t.Unlock()
		if ok {
			return value, value != ""
		}

		value = os.Getenv(env)

		t.Lock()
		t.cache[env] = value
		t.Unlock()

		return value, value != ""
	}
	return "", false
}

func (t *systemEnvironmentPropertyResolver) toEnv(key string) (string, bool) {
	if strings.HasPrefix(key, "application.") {
		prop := strings.ReplaceAll(key[len("application."):], ".", "_")
		env := strings.ToUpper(fmt.Sprintf("%s_%s", t.applicationName, prop))
		return env, true
	} else {
		return "", false
	}
}

func (t *systemEnvironmentPropertyResolver) PromptProperty(key string) (string, bool) {
	if env, ok := t.toEnv(key); ok {

		value := os.Getenv(env)
		if value == "" {
			value = sprintutils.PromptPassword(fmt.Sprintf("Enter Environment %s :", env))
		}

		t.Lock()
		t.cache[env] = value
		t.Unlock()

		return value, value != ""
	}

	return "", false
}

func (t *systemEnvironmentPropertyResolver) Environ(withValues bool) []string {
	var list []string
	t.Lock()
	defer t.Unlock()
	for k, v := range t.cache {
		if withValues {
			list = append(list, fmt.Sprintf("%s=%s", k, v))
		} else {
			list = append(list, k)
		}
	}
	return list
}

func IsPEMProperty(key string) bool {
	return strings.HasSuffix(key, ".pem") || strings.HasSuffix(key, ".key")
}

func IsPasswordProperty(key string) bool {
	return strings.HasSuffix(key, ".pwd") || strings.HasSuffix(key, ".password") || strings.HasSuffix(key, ".secret") || strings.HasSuffix(key, ".token")
}

func IsHiddenProperty(key string) bool {
	return strings.HasPrefix(key, ".")
}
