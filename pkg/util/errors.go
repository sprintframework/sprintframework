/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package util

import "github.com/pkg/errors"

func PanicToError(err *error) {
	if r := recover(); r != nil {
		switch v := r.(type) {
		case error:
			*err = v
		case string:
			*err = errors.New(v)
		default:
			*err = errors.Errorf("%v", v)
		}
	}
}

