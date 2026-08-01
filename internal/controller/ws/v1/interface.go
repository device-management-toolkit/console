package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Upgrader defines the interface for upgrading an HTTP connection to a WebSocket connection.

type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request, hdr http.Header) (*websocket.Conn, error)
}

// Redirect defines the interface for handling redirects.

type Redirect interface {
	Redirect(c *gin.Context, conn *websocket.Conn, host, mode string) error
}
