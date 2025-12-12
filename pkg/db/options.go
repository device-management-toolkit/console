/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package db

import "time"

// Option -.
type Option func(*SQL)

// MaxPoolSize -.
func MaxPoolSize(size int) Option {
	return func(c *SQL) {
		c.maxPoolSize = size
	}
}

// ConnAttempts -.
func ConnAttempts(attempts int) Option {
	return func(c *SQL) {
		c.connAttempts = attempts
	}
}

// ConnTimeout -.
func ConnTimeout(timeout time.Duration) Option {
	return func(c *SQL) {
		c.connTimeout = timeout
	}
}

// EnableForeignKeys -.
func EnableForeignKeys(value bool) Option {
	return func(c *SQL) {
		c.enableForeignKeys = value
	}
}
