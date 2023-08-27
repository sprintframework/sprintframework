/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package nat

import (
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"net"
	"sync"
	"time"
)

type implAutodiscService struct {
	what string // type of interface being auto discovered
	once sync.Once
	doit func() sprint.NatService

	mu    sync.Mutex
	found sprint.NatService
}

func startAutoDiscovery(what string, doit func() sprint.NatService) sprint.NatService {
	return &implAutodiscService{what: what, doit: doit}
}

func (t *implAutodiscService) AllowMapping() bool {
	if err := t.wait(); err != nil {
		return false
	}
	return t.found.AllowMapping()
}

func (t *implAutodiscService) AddMapping(protocol string, extport, intport int, name string, lifetime time.Duration) error {
	if err := t.wait(); err != nil {
		return err
	}
	return t.found.AddMapping(protocol, extport, intport, name, lifetime)
}

func (t *implAutodiscService) DeleteMapping(protocol string, extport, intport int) error {
	if err := t.wait(); err != nil {
		return err
	}
	return t.found.DeleteMapping(protocol, extport, intport)
}

func (t *implAutodiscService) ExternalIP() (net.IP, error) {
	if err := t.wait(); err != nil {
		return nil, err
	}
	return t.found.ExternalIP()
}

func (t *implAutodiscService) ServiceName() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.found == nil {
		return t.what
	}
	return t.found.ServiceName()
}

// wait blocks until auto-discovery has been performed.
func (t *implAutodiscService) wait() error {
	t.once.Do(func() {
		t.mu.Lock()
		t.found = t.doit()
		t.mu.Unlock()
	})
	if t.found == nil {
		return errors.Errorf("no %s router discovered", t.what)
	}
	return nil
}
