/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMaxPoolSize(t *testing.T) {
	t.Parallel()

	expectedSize := 10
	sql := &SQL{}
	MaxPoolSize(expectedSize)(sql)

	assert.Equal(t, expectedSize, sql.maxPoolSize, "MaxPoolSize() should set the maxPoolSize correctly")
}

func TestConnAttempts(t *testing.T) {
	t.Parallel()

	expectedAttempts := 3
	sql := &SQL{}
	ConnAttempts(expectedAttempts)(sql)

	assert.Equal(t, expectedAttempts, sql.connAttempts, "ConnAttempts() should set the connAttempts correctly")
}

func TestConnTimeout(t *testing.T) {
	t.Parallel()

	expectedTimeout := 5 * time.Second
	sql := &SQL{}
	ConnTimeout(expectedTimeout)(sql)

	assert.Equal(t, expectedTimeout, sql.connTimeout, "ConnTimeout() should set the connTimeout correctly")
}

func TestEnableForeignKeys(t *testing.T) {
	t.Parallel()

	expectedValue := true
	sql := &SQL{}
	EnableForeignKeys(expectedValue)(sql)

	assert.Equal(t, expectedValue, true, "EnableForeignKeys() should set the enableForeignKeys correctly")
}
