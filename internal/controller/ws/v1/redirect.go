package v1

import (
	"compress/flate"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type RedirectRoutes struct {
	d devices.Feature
	l logger.Interface
	u Upgrader
}

func RegisterRoutes(r *gin.Engine, l logger.Interface, t devices.Feature, u Upgrader) {
	rr := &RedirectRoutes{
		t,
		l,
		u,
	}
	r.GET("/relay/webrelay.ashx", rr.websocketHandler)
}

func (r *RedirectRoutes) websocketHandler(c *gin.Context) {
	host := c.Query("host")
	mode := c.Query("mode")

	r.l.Info("Websocket connection request: host=%s, mode=%s, client=%s", host, mode, c.ClientIP())

	tokenString := c.GetHeader("Sec-Websocket-Protocol")

	// validate jwt token in the Sec-Websocket-protocol header
	if !config.ConsoleConfig.Disabled {
		if tokenString == "" {
			r.l.Warn("Websocket connection rejected: missing access token (host=%s, mode=%s)", host, mode)
			http.Error(c.Writer, "request does not contain an access token", http.StatusUnauthorized)

			return
		}

		claims := &jwt.MapClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
			return []byte(config.ConsoleConfig.JWTKey), nil
		})

		if err != nil || !token.Valid {
			r.l.Warn("Websocket connection rejected: invalid access token (host=%s, mode=%s, error=%v)", host, mode, err)
			http.Error(c.Writer, "invalid access token", http.StatusUnauthorized)

			return
		}

		r.l.Debug("JWT token validated for websocket connection (host=%s, mode=%s)", host, mode)
	}

	upgrader, ok := r.u.(*websocket.Upgrader)
	if !ok {
		r.l.Debug("failed to cast Upgrader to *websocket.Upgrader")
	} else {
		upgrader.Subprotocols = []string{tokenString}
	}

	conn, err := r.u.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		r.l.Error(err, "Websocket upgrade failed (host=%s, mode=%s)", host, mode)
		http.Error(c.Writer, "Could not open websocket connection", http.StatusInternalServerError)

		return
	}

	// Optimize websocket data path for streaming; respect config compression toggle
	if config.ConsoleConfig.WSCompression {
		conn.EnableWriteCompression(true)
		_ = conn.SetCompressionLevel(flate.BestSpeed)
		r.l.Debug("Websocket compression enabled (host=%s, mode=%s)", host, mode)
	} else {
		conn.EnableWriteCompression(false)
		_ = conn.SetCompressionLevel(flate.NoCompression)
		r.l.Debug("Websocket compression disabled (host=%s, mode=%s)", host, mode)
	}

	r.l.Info("Websocket connection opened successfully (host=%s, mode=%s)", host, mode)

	err = r.d.Redirect(c, conn, host, mode)
	if err != nil {
		r.l.Error(err, "Redirect failed (host=%s, mode=%s)", host, mode)
		errorResponse(c, http.StatusInternalServerError, "redirect failed")
	} else {
		r.l.Info("Websocket connection closed normally (host=%s, mode=%s)", host, mode)
	}
}
