package v1

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/device-management-toolkit/console/config"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var (
	ErrLogin                   = consoleerrors.CreateConsoleError("LoginHandler")
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrCSRFTokenMissing        = errors.New("CSRF token missing")
	ErrCSRFTokenMismatch       = errors.New("CSRF token mismatch")
)

const (
	sessionCookieName = "session"
	csrfCookieName    = "XSRF-TOKEN"
	csrfHeaderName    = "X-XSRF-TOKEN"
	csrfTokenByteLen  = 32
)

type LoginRoute struct {
	Config   *config.Config
	Verifier *oidc.IDTokenVerifier
}

// NewLoginRoute creates a new login route.
func NewLoginRoute(configData *config.Config) *LoginRoute {
	lr := &LoginRoute{
		Config: configData,
	}

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

// generateCSRFToken creates a cryptographically random URL-safe base64 token.
func generateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenByteLen)

	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateCSRFToken: %w", err)
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

// setAuthCookies writes the HttpOnly session cookie and the readable XSRF-TOKEN
// cookie after a successful authentication. The session cookie carries the JWT
// and is never accessible to JavaScript. The XSRF-TOKEN cookie is intentionally
// readable so Angular's built-in XSRF interceptor can attach it as a request
// header, implementing the double-submit cookie CSRF pattern.
func (lr LoginRoute) setAuthCookies(c *gin.Context, tokenString string) error {
	csrfToken, err := generateCSRFToken()
	if err != nil {
		return err
	}

	secure := lr.Config.HTTP.TLS.Enabled
	maxAge := int(config.ConsoleConfig.JWTExpiration.Seconds())

	// HttpOnly=true — JavaScript cannot read this cookie.
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    tokenString,
		MaxAge:   maxAge,
		Path:     "/",
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// HttpOnly=false — intentionally readable by Angular's XSRF interceptor.
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     csrfCookieName,
		Value:    csrfToken,
		MaxAge:   maxAge,
		Path:     "/",
		Secure:   secure,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	})

	return nil
}

// clearAuthCookies expires both session cookies, effectively logging out a
// browser client.
func (lr LoginRoute) clearAuthCookies(c *gin.Context) {
	secure := lr.Config.HTTP.TLS.Enabled

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     csrfCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Secure:   secure,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	})
}

// validateCSRF implements the double-submit cookie pattern: the X-XSRF-TOKEN
// request header must be present and must match the XSRF-TOKEN cookie value.
// This is only checked for cookie-authenticated requests; requests carrying an
// Authorization: Bearer header are inherently CSRF-resistant and skip this check.
func validateCSRF(c *gin.Context) error {
	csrfHeader := c.GetHeader(csrfHeaderName)
	if csrfHeader == "" {
		return ErrCSRFTokenMissing
	}

	csrfCookie, err := c.Cookie(csrfCookieName)
	if err != nil || csrfCookie == "" {
		return ErrCSRFTokenMissing
	}

	if csrfHeader != csrfCookie {
		return ErrCSRFTokenMismatch
	}

	return nil
}

// isStateChangingMethod returns true for HTTP methods that modify server state
// and therefore require CSRF validation when using cookie-based auth.
func isStateChangingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return true
	default:
		return false
	}
}

// Login checks configured credentials and returns a JWT token for basic auth.
// It also sets an HttpOnly session cookie and a readable XSRF-TOKEN cookie for
// browser clients.
func (lr LoginRoute) Login(c *gin.Context) {
	var creds dto.Credentials

	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: "invalid request"})

		return
	}

	lr.handleBasicAuth(creds, c)
}

func (lr LoginRoute) handleBasicAuth(creds dto.Credentials, c *gin.Context) {
	if creds.Username != lr.Config.AdminUsername || creds.Password != lr.Config.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{errorKey: "invalid credentials", messageKey: "Incorrect Username and/or Password!"})

		return
	}

	// Create JWT token.
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

	// Set HttpOnly session cookie + readable XSRF-TOKEN cookie for browser clients.
	// The JSON body still carries the token for backward-compatible API clients
	// (rpc-go, Postman, scripts) that use the Authorization: Bearer header.
	if err := lr.setAuthCookies(c, tokenString); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: "could not create session"})

		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// Logout clears the session and XSRF-TOKEN cookies, ending the browser session.
// API clients using Authorization: Bearer are unaffected; their tokens remain
// valid until expiry (stateless JWT — no server-side revocation).
func (lr LoginRoute) Logout(c *gin.Context) {
	lr.clearAuthCookies(c)
	c.Status(http.StatusNoContent)
}

// JWTAuthMiddleware validates the caller's identity. It accepts a token from
// either the Authorization: Bearer header (API clients, rpc-go, OAuth) or from
// the HttpOnly session cookie (browser clients). When the cookie path is taken,
// CSRF validation is enforced on state-changing methods (POST/PUT/DELETE/PATCH)
// using the double-submit cookie pattern.
func (lr LoginRoute) JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		usingCookie := false

		if tokenString == "" {
			// Bearer header absent — fall back to HttpOnly session cookie.
			cookieVal, err := c.Cookie(sessionCookieName)
			if err != nil || cookieVal == "" {
				c.JSON(http.StatusUnauthorized, gin.H{errorKey: "request does not contain an access token"})
				c.Abort()

				return
			}

			tokenString = cookieVal
			usingCookie = true
		}

		// Validate the token — OIDC verifier when a client ID is configured,
		// otherwise verify the HS256 JWT we issued ourselves.
		if config.ConsoleConfig.ClientID != "" {
			if _, err := lr.Verifier.Verify(c.Request.Context(), tokenString); err != nil {
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

		// CSRF check — only for cookie-authenticated, state-changing requests.
		// Authorization: Bearer requests are inherently CSRF-resistant.
		if usingCookie && isStateChangingMethod(c.Request.Method) {
			if err := validateCSRF(c); err != nil {
				c.JSON(http.StatusForbidden, gin.H{errorKey: "CSRF validation failed"})
				c.Abort()

				return
			}
		}

		c.Next()
	}
}
