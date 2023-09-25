/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintserver_test

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

var robotsTxt = "\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x0a\x2d\x4e\x2d\xd2\x4d\x4c\x4f\xcd\x2b\xb1\x52\xd0\xe2\xe5\x72\xc9\x2c\x4e\xcc\xc9\xc9\x2f\xb7\x52\xd0\x4f\x2c\xc8\xd4\xe7\xe5\x02\x04\x00\x00\xff\xff\x25\xc9\xc7\x6c\x20\x00\x00\x00"

func TestGzipUnpack(t *testing.T) {

	fmt.Printf("compressed len %d\n", len(robotsTxt))

	var plain bytes.Buffer
	zr, err := gzip.NewReader(bytes.NewReader([]byte(robotsTxt)))
	require.NoError(t, err)

	n, err := io.Copy(&plain, zr)
	require.NoError(t, err)
	err = zr.Close()
	require.NoError(t, err)

	println(n)
	plaintText := string(plain.Bytes())
	println(plaintText)

}

