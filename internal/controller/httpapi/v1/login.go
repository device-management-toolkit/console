package v1

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

const (
	loginRateBurst         = 5                // loginRateBurst is the maximum burst of login attempts allowed before throttling.
	loginRateWindow        = 12 * time.Second // loginRateWindow is the token-refill interval: one token per window → 5 per minute.
	limiterTTL             = 15 * time.Minute // limiterTTL is how long an idle per-IP entry is kept before the cleanup goroutine removes it.
	limiterCleanupInterval = 5 * time.Minute  // limiterCleanupInterval is how often the cleanup goroutine sweeps stale entries.
)

// loginRatePerIP is the sustained token-refill rate: one token every loginRateWindow → 5 per minute.
var loginRatePerIP = rate.Every(loginRateWindow) //nolint:gochecknoglobals // rate.Limit is a float64; cannot be const

var (
	ErrLogin                   = consoleerrors.CreateConsoleError("LoginHandler")
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
)

// ipEntry pairs a rate limiter with the last time it was accessed, enabling
// stale-entry cleanup without tracking a separate timestamp map.
type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type LoginRoute struct {
	Config   *config.Config
	Verifier *oidc.IDTokenVerifier
	mu       sync.Mutex
	limiters map[string]*ipEntry
}

// NewLoginRoute creates a new login route with per-IP rate limiting enabled.
func NewLoginRoute(configData *config.Config) *LoginRoute {
	lr := &LoginRoute{
		Config:   configData,
		limiters: make(map[string]*ipEntry),
	}

	go lr.cleanupLoop()

	if config.ConsoleConfig.ClientID != "" {
		ctx := context.Background()

		if config.ConsoleConfig.TLSSkipVerify {
			transport, _ := http.DefaultTransport.(*http.Transport)
			transport = transport.Clone()
			transport.TLSClientConfig = &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true, //nolint:gosec // operator opted in via auth.tlsSkipVerify to trust self-signed IdP
			}

			ctx = oidc.ClientContext(ctx, &http.Client{Transport: transport})
		}

		provider, err := oidc.NewProvider(ctx, config.ConsoleConfig.Issuer)
		if err != nil {
			return nil
		}

		lr.Verifier = provider.Verifier(&oidc.Config{
			ClientID: config.ConsoleConfig.ClientID,
		})
	}

	return lr
}

// getLimiter returns (creating if necessary) the rate limiter for the given IP.
func (lr *LoginRoute) getLimiter(ip string) *rate.Limiter {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	// Lazy-initialize so direct struct construction in tests still works.
	if lr.limiters == nil {
		lr.limiters = make(map[string]*ipEntry)
	}

	entry, ok := lr.limiters[ip]
	if !ok {
		entry = &ipEntry{limiter: rate.NewLimiter(loginRatePerIP, loginRateBurst)}
		lr.limiters[ip] = entry
	}

	entry.lastSeen = time.Now()

	return entry.limiter
}

// cleanupLoop removes stale per-IP limiters on a fixed interval.
// It runs as a background goroutine for the lifetime of the server.
func (lr *LoginRoute) cleanupLoop() {
	ticker := time.NewTicker(limiterCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		lr.mu.Lock()

		for ip, entry := range lr.limiters {
			if time.Since(entry.lastSeen) > limiterTTL {
				delete(lr.limiters, ip)
			}
		}

		lr.mu.Unlock()
	}
}

// Login checks configured credentials and returns a JWT token for basic auth.
func (lr *LoginRoute) Login(c *gin.Context) {
	if !lr.getLimiter(c.ClientIP()).Allow() {
		c.Header("Retry-After", "60")
		c.JSON(http.StatusTooManyRequests, gin.H{
			errorKey:   "too many requests",
			messageKey: "Too many login attempts. Please try again later.",
		})

		return
	}

	var creds dto.Credentials

	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: "invalid request"})

		return
	}

	lr.handleBasicAuth(creds, c)
}

func (lr *LoginRoute) handleBasicAuth(creds dto.Credentials, c *gin.Context) {
	if creds.Username != lr.Config.AdminUsername || creds.Password != lr.Config.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{errorKey: "invalid credentials", messageKey: "Incorrect Username and/or Password!"})

		return
	}

	// Create JWT token
	expirationTime := time.Now().Add(config.ConsoleConfig.JWTExpiration)
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		Issuer:    config.ConsoleConfig.Issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(lr.Config.JWTKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: "could not create token"})

		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// JWT Middleware
func (lr *LoginRoute) JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		tokenString = strings.Replace(tokenString, "Bearer ", "", 1)

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{errorKey: "request does not contain an access token"})
			c.Abort()

			return
		}

		// if clientID is set, use the oidc verifier
		if config.ConsoleConfig.ClientID != "" {
			_, err := lr.Verifier.Verify(c.Request.Context(), tokenString)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{errorKey: "invalid access token"})
				c.Abort()

				return
			}
		} else {
			claims := &jwt.MapClaims{}

			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, token.Header["alg"])
				}

				return []byte(lr.Config.JWTKey), nil
			})

			if err != nil || !token.Valid {
				c.JSON(http.StatusUnauthorized, gin.H{errorKey: "invalid access token"})
				c.Abort()

				return
			}
		}

		c.Next()
	}
}
