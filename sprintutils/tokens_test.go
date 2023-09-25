/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintutils_test

import (
	"github.com/sprintframework/sprintframework/sprintutils"
	"github.com/stretchr/testify/require"
	"math"
	"math/rand"
	"testing"
)

func TestLongId(t *testing.T) {

	id, err := sprintutils.GenerateLongId()
	require.NoError(t, err)

	value, err := sprintutils.DecodeLongId(id)
	require.NoError(t, err)

	require.Equal(t, id, sprintutils.EncodeLongId(value))

}

func TestShortId(t *testing.T) {

	for i := 0; i < 100; i++ {
		n := rand.Uint64() % uint64(math.Pow10(i/5))
		str := sprintutils.EncodeId(n)
		actual, err := sprintutils.DecodeId(str)
		require.NoError(t, err)
		require.Equal(t, n, actual)
	}

}

func TestShowId(t *testing.T) {
	num, _ := sprintutils.DecodeId("s00001")
	println(num)

	println(sprintutils.EncodeId(num+1))
}
