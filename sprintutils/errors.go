/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintutils

import (
	"github.com/pkg/errors"
	"runtime/debug"
)

func PanicToError(err *error) {
	if r := recover(); r != nil {
		*err = errors.Errorf("%v, %s", r, debug.Stack())
	}
}

