package v1

import (
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
	tokenString := c.GetHeader("Sec-Websocket-Protocol")

	// validate jwt token in the Sec-Websocket-protocol header
	if !config.ConsoleConfig.Disabled {
		if tokenString == "" {
			http.Error(c.Writer, "request does not contain an access token", http.StatusUnauthorized)

			return
		}

		claims := &jwt.MapClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
			return []byte(config.ConsoleConfig.Auth.JWTKey), nil
		})

		if err != nil || !token.Valid {
			http.Error(c.Writer, "invalid access token", http.StatusUnauthorized)

			return
		}
	}

	upgrader, ok := r.u.(*websocket.Upgrader)
	if !ok {
		r.l.Debug("failed to cast Upgrader to *websocket.Upgrader")
	} else {
		upgrader.Subprotocols = []string{tokenString}
	}

	conn, err := r.u.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		http.Error(c.Writer, "Could not open websocket connection", http.StatusInternalServerError)

		return
	}

	r.l.Info("Websocket connection opened")

	err = r.d.Redirect(c, conn, c.Query("host"), c.Query("mode"))
	if err != nil {
		r.l.Error(err, "http - devices - v1 - redirect")
		errorResponse(c, http.StatusInternalServerError, "redirect failed")
	}
}
