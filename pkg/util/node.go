/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package util

import "fmt"

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

