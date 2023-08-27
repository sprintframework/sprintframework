/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package nat

import (
	"errors"
	"github.com/sprintframework/sprint"
	"net"
	"time"
)

var (
	ErrNoNatService = errors.New("no nat service")
)

type implNonatService struct {
}

func NoNatService() sprint.NatService {
	return &implNonatService{}
}

func (t *implNonatService) ServiceName() string {
	return "no_nat"
}

func (t *implNonatService) AllowMapping() bool {
	return false
}

func (t *implNonatService) AddMapping(protocol string, extport, intport int, name string, lifetime time.Duration) error {
	return nil
}

func (t *implNonatService) DeleteMapping(protocol string, extport, intport int) error {
	return nil
}

func (t *implNonatService) ExternalIP() (net.IP, error) {
	return nil, ErrNoNatService
}


