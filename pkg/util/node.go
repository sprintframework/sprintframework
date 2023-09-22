/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package util

import (
	"fmt"
	"github.com/pkg/errors"
	"net"
)

/**
Formats unique name of the node by adding sequence number of it to name.
 */

func AppendNodeSequence(name string, seq int) string {
	if seq == 0 {
		return name
	} else {
		return fmt.Sprintf("%s-%d", name, seq)
	}
}

func AppendNodeName(name string, next string) string {
	return fmt.Sprintf("%s-%s", name, next)
}

func ParseAndAdjustTCPAddr(address string, seq int) (*net.TCPAddr, error) {

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.Errorf("empty port in address '%s', %v", address, err)
	}
	if host == "" {
		// empty host means all IPs
		host = "0.0.0.0"
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	// Resolve the address
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, errors.Errorf("invalid address '%s', %v", addr, err)
	}

	tcpAddr.Port += seq

	return tcpAddr, nil

}