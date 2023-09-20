/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package util

import (
	"fmt"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

/**
Formats unique name of the node by adding sequence number of it to application name.
 */

func FormatNodeName(applicationName string, node int) string {
	if node == 0 {
		return applicationName
	} else {
		return fmt.Sprintf("%s-%d", applicationName, node)
	}
}

func AdjustPortNumberInAddress(addr string, seq int) (result string, err error) {
	if seq == 0 {
		return addr, nil
	}
	parts := strings.Split(addr, ":")
	if len(parts) > 0 {
		lastIndex := len(parts)-1
		parts[lastIndex], err = AdjustPortNumber(parts[lastIndex], seq)
		if err != nil {
			return
		}
		return strings.Join(parts, ":"), nil
	}
	return addr, nil
}

func AdjustPortNumber(port string, seq int) (string, error) {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return "", errors.Errorf("invalid port number string '%s', %v", port, err)
	}
	if portNum == 0 {
		// do not adjust zero port number, because it is the any one
		return port, nil
	}
	return strconv.Itoa(portNum + seq), nil
}
