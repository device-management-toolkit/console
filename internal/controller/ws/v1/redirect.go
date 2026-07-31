package v1

import (
	"compress/flate"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var ErrUnexpectedSigningMethod = errors.New("unexpected signing method")

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

	// validate the jwt token in the Sec-Websocket-protocol header
	if !r.validateRedirectionToken(c, tokenString) {
		return
	}

	upgrader, ok := r.u.(*websocket.Upgrader)
	if !ok {
		r.l.Debug("failed to cast Upgrader to *websocket.Upgrader")
	} else {
		upgrader.Subprotocols = []string{tokenString}
	}

	// KVM_TIMING: Measure WebSocket upgrade duration
	upgradeStart := time.Now()
	conn, err := r.u.Upgrade(c.Writer, c.Request, nil)
	upgradeDuration := time.Since(upgradeStart)
	devices.RecordWebsocketUpgrade(upgradeDuration)
	r.l.Debug("KVM_TIMING: WebSocket upgrade", "duration_ms", upgradeDuration.Milliseconds())

	if err != nil {
		http.Error(c.Writer, "Could not open websocket connection", http.StatusInternalServerError)

		return
	}

	// Optimize websocket data path for streaming; respect config compression toggle
	if config.ConsoleConfig.WSCompression {
		conn.EnableWriteCompression(true)
		_ = conn.SetCompressionLevel(flate.BestSpeed)
	} else {
		conn.EnableWriteCompression(false)
		_ = conn.SetCompressionLevel(flate.NoCompression)
	}

	r.l.Info("Websocket connection opened")

	// KVM_TIMING: Measure total connection time
	totalStart := time.Now()
	err = r.d.Redirect(c, conn, c.Query("host"), c.Query("mode"))
	totalDuration := time.Since(totalStart)
	devices.RecordTotalConnection(totalDuration, c.Query("mode"))
	r.l.Debug("KVM_TIMING: Total connection time", "duration_ms", totalDuration.Milliseconds(), "mode", c.Query("mode"))

	if err != nil {
		r.l.Error(err, "http - devices - v1 - redirect")
		errorResponse(c, http.StatusInternalServerError, "redirect failed")
	}
}

// validateRedirectionToken checks the JWT and that its deviceId matches the host.
func (r *RedirectRoutes) validateRedirectionToken(c *gin.Context, tokenString string) bool {
	if config.ConsoleConfig.Disabled {
		return true
	}

	if tokenString == "" {
		http.Error(c.Writer, "request does not contain an access token", http.StatusUnauthorized)

		return false
	}

	claims := &jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, token.Header["alg"])
		}

		return []byte(config.ConsoleConfig.JWTKey), nil
	})

	if err != nil || !token.Valid {
		http.Error(c.Writer, "invalid access token", http.StatusUnauthorized)

		return false
	}

	// deviceId must be present and match host; blocks other-device and login tokens.
	// GUIDs are case-insensitive, so match them that way to avoid false rejections.
	deviceID, _ := (*claims)["deviceId"].(string)
	if deviceID == "" || !strings.EqualFold(deviceID, c.Query("host")) {
		r.l.Warn("redirection token not authorized for requested device", "host", c.Query("host"))
		http.Error(c.Writer, "token not authorized for this device", http.StatusForbidden)

		return false
	}

	return true
}
