/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintserver

import (
	rt "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/url"
	"testing"
)

func TestHttpMux(t *testing.T) {

	mux := http.NewServeMux()
	api := rt.NewServeMux()
	mux.Handle("/api/", api)

	u, err := url.Parse("http://localhost:8443/api/")
	require.NoError(t, err)

	req := &http.Request{
		Method:     "GET",
		URL:        u,
		Host:       "localhost",
		RequestURI: "/api/",
	}

	handler, foundPattern := mux.Handler(req)
	require.Equal(t, "/api/", foundPattern)
	require.Equal(t, handler, api)

}

func TestHttpMuxRewrite(t *testing.T) {
	u, err := url.Parse("http://localhost:8443/")
	require.NoError(t, err)
	u.Path = "/index.html"
	require.Equal(t, u.RequestURI(), "/index.html")
}
