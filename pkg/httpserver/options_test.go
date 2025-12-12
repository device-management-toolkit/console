/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package httpserver

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPort(t *testing.T) {
	t.Parallel()

	s := &Server{server: &http.Server{
		ReadHeaderTimeout: 1 * time.Second,
	}}
	opt := Port("localhost", "8080")
	opt(s)

	expectedAddr := net.JoinHostPort("localhost", "8080")
	assert.Equal(t, expectedAddr, s.server.Addr, "Port() should set the correct server address")
}

func TestReadTimeout(t *testing.T) {
	t.Parallel()

	s := &Server{server: &http.Server{
		ReadHeaderTimeout: 1 * time.Second,
	}}
	timeout := 5 * time.Second
	opt := ReadTimeout(timeout)
	opt(s)

	assert.Equal(t, timeout, s.server.ReadTimeout, "ReadTimeout() should set the correct read timeout")
}

func TestWriteTimeout(t *testing.T) {
	t.Parallel()

	s := &Server{server: &http.Server{
		ReadHeaderTimeout: 1 * time.Second,
	}}
	timeout := 5 * time.Second
	opt := WriteTimeout(timeout)
	opt(s)

	assert.Equal(t, timeout, s.server.WriteTimeout, "WriteTimeout() should set the correct write timeout")
}

func TestShutdownTimeout(t *testing.T) {
	t.Parallel()

	s := &Server{}
	timeout := 5 * time.Second
	opt := ShutdownTimeout(timeout)
	opt(s)

	assert.Equal(t, timeout, s.shutdownTimeout, "ShutdownTimeout() should set the correct shutdown timeout")
}
