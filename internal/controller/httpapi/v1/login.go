package v1

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/secrets"
)

const (
	// sessionCookiePath covers both the UI (/) and the API (/api).
	sessionCookiePath = "/"

	authorizationHeader         = "Authorization"
	bearerPrefix                = "Bearer "
	incorrectCredentialsMessage = "Incorrect Username and/or Password!"
	loginFailureError           = "invalid credentials"

	// credentialTagClaim binds a keyring-mode token to the stored password hash.
	credentialTagClaim = "cred"
)

var (
	ErrLogin                   = consoleerrors.CreateConsoleError("LoginHandler")
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")

	// passwordChangeMu serializes password changes.
	passwordChangeMu sync.Mutex
)

type LoginRoute struct {
	Config   *config.Config
	Verifier *oidc.IDTokenVerifier
	// Keyring is the OS keyring holding the administrator credentials. It is
	// set only for AUTH_SECRET_BACKEND=keyring; other backends leave it nil
	// and keep their existing behavior.
	Keyring security.Storager
}

// NewVersionRoute creates a new version route
func NewLoginRoute(configData *config.Config) *LoginRoute {
	lr := &LoginRoute{
		Config: configData,
	}

	if configData.SecretBackend == "keyring" && !configData.Disabled && configData.ClientID == "" {
		lr.Keyring = security.NewKeyRingStorage(secrets.KeyringServiceName)
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

// Login checks configured credentials and returns a JWT token for basic auth
func (lr LoginRoute) Login(c *gin.Context) {
	var creds dto.Credentials

	// Not BindJSON: the error is deliberately discarded so login never reports
	// why the body was rejected.
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: "invalid request"})

		return
	}

	lr.handleBasicAuth(creds, c)
}

func (lr LoginRoute) handleBasicAuth(creds dto.Credentials, c *gin.Context) {
	// Nothing verifies tokens with auth off, and a signed one would still pass once auth is enabled.
	if lr.Config.Disabled {
		c.JSON(http.StatusOK, gin.H{tokenKey: ""})

		return
	}

	if lr.Config.ClientID != "" {
		c.JSON(http.StatusUnauthorized, gin.H{errorKey: loginFailureError, messageKey: incorrectCredentialsMessage})

		return
	}

	adminUsername, adminPassword := lr.adminCredentials()

	if adminPassword == "" {
		c.JSON(http.StatusUnauthorized, gin.H{errorKey: loginFailureError, messageKey: incorrectCredentialsMessage})

		return
	}

	var passwordMatches bool

	switch lr.Config.SecretBackend {
	case "config":
		passwordMatches = subtle.ConstantTimeCompare([]byte(creds.Password), []byte(adminPassword)) == 1
	case "keyring":
		passwordMatches = secrets.VerifyPBKDF2Hash(adminPassword, creds.Password)
	}

	usernameMatches := subtle.ConstantTimeCompare([]byte(creds.Username), []byte(adminUsername)) == 1

	if !passwordMatches || !usernameMatches {
		c.JSON(http.StatusUnauthorized, gin.H{errorKey: loginFailureError, messageKey: incorrectCredentialsMessage})

		return
	}

	// Create JWT token
	expirationTime := time.Now().Add(config.ConsoleConfig.JWTExpiration)
	claims := jwt.MapClaims{
		"exp": expirationTime.Unix(),
	}

	// Keyring mode only: a password change replaces the stored hash, so the
	// tag no longer matches and every earlier token is rejected.
	if lr.Keyring != nil {
		claims[credentialTagClaim] = lr.credentialTag(adminPassword)
	}

	// Conditional so the claim set still matches what callers saw before.
	if config.ConsoleConfig.Issuer != "" {
		claims["iss"] = config.ConsoleConfig.Issuer
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(lr.Config.JWTKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: errTokenCreation})

		return
	}

	setSessionCookies(c, tokenString, expirationTime)

	// Token stays in the body for bearer clients, which ignore Set-Cookie.
	c.JSON(http.StatusOK, gin.H{tokenKey: tokenString})
}

// Logout expires the session cookies. Public, so an already-expired session can
// still clear its own. Not revocation: the JWT stays valid until it expires.
func (lr LoginRoute) Logout(c *gin.Context) {
	clearSessionCookies(c)
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	c.JSON(http.StatusOK, gin.H{messageKey: "logged out"})
}

// JWTAuthMiddleware accepts either the Authorization header (REST clients) or
// the session cookie (browser). The header wins, so REST clients are unchanged.
func (lr LoginRoute) JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := resolveToken(c)

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{errorKey: "request does not contain an access token"})
			c.Abort()

			return
		}

		if !lr.verifyToken(c, tokenString) {
			c.JSON(http.StatusUnauthorized, gin.H{errorKey: "invalid access token"})
			c.Abort()

			return
		}

		c.Next()
	}
}

// resolveToken prefers the Authorization header, falling back to the cookie.
func resolveToken(c *gin.Context) string {
	if header := c.GetHeader(authorizationHeader); header != "" {
		return strings.Replace(header, bearerPrefix, "", 1)
	}

	if !cookieAuthEnabled() {
		return ""
	}

	cookie, err := c.Cookie(sessionCookieName())
	if err != nil {
		return ""
	}

	return cookie
}

// verifyToken reports whether the token is valid.
func (lr LoginRoute) verifyToken(c *gin.Context, tokenString string) bool {
	// if clientID is set, use the oidc verifier
	if config.ConsoleConfig.ClientID != "" {
		_, err := lr.Verifier.Verify(c.Request.Context(), tokenString)

		return err == nil
	}

	claims := &jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, token.Header["alg"])
		}

		return []byte(lr.Config.JWTKey), nil
	})

	if err != nil || !token.Valid {
		return false
	}

	if lr.Keyring == nil {
		return true
	}

	tag, _ := (*claims)[credentialTagClaim].(string)

	storedHash, err := lr.Keyring.GetKeyValue(secrets.KeyringAdminPasswordKey)
	if err != nil || tag == "" {
		return false
	}

	return hmac.Equal([]byte(tag), []byte(lr.credentialTag(storedHash)))
}

// adminCredentials returns the administrator username and stored password.
// Keyring mode reads the keyring on every call, so a password change takes
// effect immediately.
func (lr LoginRoute) adminCredentials() (username, password string) {
	if lr.Keyring == nil {
		return lr.Config.AdminUsername, lr.Config.AdminPassword
	}

	username, usernameErr := lr.Keyring.GetKeyValue(secrets.KeyringAdminUsernameKey)
	password, passwordErr := lr.Keyring.GetKeyValue(secrets.KeyringAdminPasswordKey)

	if usernameErr != nil || passwordErr != nil {
		return "", ""
	}

	return username, password
}

// credentialTag is a keyed, non-reversible tag of the stored password hash.
func (lr LoginRoute) credentialTag(storedHash string) string {
	mac := hmac.New(sha256.New, []byte(lr.Config.JWTKey))
	mac.Write([]byte(storedHash))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// ChangePassword replaces the keyring administrator password after verifying
// the current one. Registered only when Keyring is set.
func (lr LoginRoute) ChangePassword(c *gin.Context) {
	var req dto.PasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: "invalid request"})

		return
	}

	switch {
	case req.CurrentPassword == "" || req.NewPassword == "" || req.ConfirmPassword == "":
		c.JSON(http.StatusBadRequest, gin.H{errorKey: "current password, new password, and confirmation are required"})

		return
	case req.NewPassword != req.ConfirmPassword:
		c.JSON(http.StatusBadRequest, gin.H{errorKey: "password confirmation does not match"})

		return
	case !secrets.IsStrongAdminPassword(req.NewPassword):
		c.JSON(http.StatusBadRequest, gin.H{errorKey: "new password must be at least 8 characters and contain " +
			"a lowercase letter, an uppercase letter, a digit, and a symbol"})

		return
	}

	passwordChangeMu.Lock()
	defer passwordChangeMu.Unlock()

	storedHash, err := lr.Keyring.GetKeyValue(secrets.KeyringAdminPasswordKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: "password storage is unavailable"})

		return
	}

	if !secrets.VerifyPBKDF2Hash(storedHash, req.CurrentPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{errorKey: loginFailureError, messageKey: incorrectCredentialsMessage})

		return
	}

	if req.NewPassword == req.CurrentPassword {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: "new password must differ from current password"})

		return
	}

	newHash, err := secrets.GeneratePBKDF2Hash(req.NewPassword)
	if err == nil {
		err = lr.Keyring.SetKeyValue(secrets.KeyringAdminPasswordKey, newHash)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: "password storage is unavailable"})

		return
	}

	c.JSON(http.StatusOK, dto.PasswordChangeResponse{Message: "password changed"})
}

// cookieAuthEnabled reports whether session cookies are in use. An unconfigured
// process issues none.
func cookieAuthEnabled() bool {
	if config.ConsoleConfig == nil {
		return false
	}

	return config.ConsoleConfig.CookieAuthEnabled()
}

func sessionCookieName() string {
	if config.ConsoleConfig.CookieName != "" {
		return config.ConsoleConfig.CookieName
	}

	return config.DefaultSessionCookieName
}

func cookieSameSite() http.SameSite {
	switch strings.ToLower(config.ConsoleConfig.CookieSameSite) {
	case "none":
		return http.SameSiteNoneMode
	case "lax":
		return http.SameSiteLaxMode
	default:
		return http.SameSiteStrictMode
	}
}

// setSessionCookies issues the JWT as an HttpOnly cookie, so the browser need
// not persist it in Web Storage.
func setSessionCookies(c *gin.Context, tokenString string, expiresAt time.Time) {
	if !cookieAuthEnabled() {
		return
	}

	writeSessionCookie(c, tokenString, int(time.Until(expiresAt).Seconds()))
}

// clearSessionCookies expires the cookie, even when cookie auth is now off, so
// one issued earlier can still be cleaned up.
func clearSessionCookies(c *gin.Context) {
	writeSessionCookie(c, "", -1)
}

func writeSessionCookie(c *gin.Context, tokenString string, maxAge int) {
	if config.ConsoleConfig == nil {
		return
	}

	secure := config.ConsoleConfig.CookieSecure
	sameSite := cookieSameSite()

	// Browsers drop SameSite=None cookies that are not Secure.
	if sameSite == http.SameSiteNoneMode {
		secure = true
	}

	//nolint:gosec // G124: Secure and SameSite are set from config above; the linter cannot see through the variables
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookieName(),
		Value:    tokenString,
		Path:     sessionCookiePath,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: true,
		SameSite: sameSite,
	})
}
