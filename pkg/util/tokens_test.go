/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package util_test

import (
	"github.com/sprintframework/sprintframework/pkg/util"
	"github.com/stretchr/testify/require"
	"math"
	"math/rand"
	"testing"
)

func TestLongId(t *testing.T) {

	id, err := util.GenerateLongId()
	require.NoError(t, err)

	value, err := util.DecodeLongId(id)
	require.NoError(t, err)

	require.Equal(t, id, util.EncodeLongId(value))

}

func TestShortId(t *testing.T) {

	for i := 0; i < 100; i++ {
		n := rand.Uint64() % uint64(math.Pow10(i/5))
		str := util.EncodeId(n)
		actual, err := util.DecodeId(str)
		require.NoError(t, err)
		require.Equal(t, n, actual)
	}

}

func TestShowId(t *testing.T) {
	num, _ := util.DecodeId("s00001")
	println(num)

	println(util.EncodeId(num+1))
}
