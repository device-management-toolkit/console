/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package app

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/gorilla/websocket"
)

// DB is an interface for database operations.
type DB interface {
	Close() error
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// HTTPServer is an interface for the HTTP server.
type HTTPServer interface {
	Notify() <-chan error
	Shutdown() error
	Start() error
}

// WebSocketUpgrader is an interface for WebSocket upgrading.
type WebSocketUpgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error)
}
