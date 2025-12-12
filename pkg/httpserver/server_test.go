/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package httpserver

import (
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) { //nolint:paralleltest // httpserver can't be bind to multiple ports at the same time for tests
	handler := http.NewServeMux()
	s := New(handler, Port("localhost", "9090")) // Use a different port

	defer s.server.Close()

	assert.Equal(t, handler, s.server.Handler, "expected handler to be set")
	assert.Equal(t, _defaultReadTimeout, s.server.ReadTimeout, "expected read timeout to be set correctly")
	assert.Equal(t, _defaultWriteTimeout, s.server.WriteTimeout, "expected write timeout to be set correctly")
	assert.Equal(t, net.JoinHostPort("localhost", "9090"), s.server.Addr, "expected addr to be set correctly")
	assert.Equal(t, _defaultShutdownTimeout, s.shutdownTimeout, "expected shutdown timeout to be set correctly")
}
