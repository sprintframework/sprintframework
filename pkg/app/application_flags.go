/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package app

import (
	"flag"
	"fmt"
	"github.com/sprintframework/sprint"
	"strings"
)

type implApplicationFlags struct {

	priority int

	daemon     *bool
	verbose    *bool

	properties  keyValueFlags

}

type keyValueFlags map[string]string

func NewKeyValueFlags() keyValueFlags {
	return make(map[string]string)
}

func (f *keyValueFlags) String() string {
	return "application properties key=value"
}

func (f *keyValueFlags) Set(value string) error {
	if *f == nil {
		*f = make(map[string]string)
	}
	i := strings.IndexByte(value, '=')
	if i == -1 {
		(*f)[value] = ""
	} else {
		(*f)[value[:i]] = value[i+1:]
	}
	return nil
}

func ApplicationFlags(priority int) sprint.ApplicationFlags {
	return &implApplicationFlags{
		priority: priority,
		properties:  NewKeyValueFlags(),
	}
}

func (t *implApplicationFlags) String() string {
	return fmt.Sprintf("ApplicationFlags{%v,%d}", t.RegisterServerArgs(make([]string, 0, 10)), t.priority)
}

func (t *implApplicationFlags) RegisterFlags(fs *flag.FlagSet) {
	t.daemon = fs.Bool("d", false, "Run as daemon")
	t.verbose = fs.Bool("v", false, "Verbose debug information")
	fs.Var(&t.properties, "p", "Application override properties key=value")
}

func (t *implApplicationFlags) RegisterServerArgs(args []string) []string {

	if t.Verbose() {
		args = append(args, "-v")
	}

	for k, v := range t.properties {
		if k != "" {
			args = append(args, "-p", fmt.Sprintf("%s=%s", k, v))
		}
	}

	return args
}

func (t *implApplicationFlags) Daemon() bool {
	if t.daemon != nil {
		return *t.daemon
	}
	return false
}

func (t *implApplicationFlags) Verbose() bool {
	if t.verbose != nil {
		return *t.verbose
	}
	return false
}

func (t *implApplicationFlags) Properties() map[string]string {
	return t.properties
}

func (t *implApplicationFlags) Priority() int {
	return t.priority
}

func (t *implApplicationFlags) GetProperty(key string) (value string, ok bool) {
	if t.properties == nil {
		return "", false
	}
	value, ok = t.properties[key]
	return
}

