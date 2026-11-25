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
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/authsetup"
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
	keyringAdminPassword        = "admin-password-hash"
	secretBackendKeyring        = "keyring"
	credentialTagClaim          = "cred"
)

var (
	ErrLogin                   = consoleerrors.CreateConsoleError("LoginHandler")
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	errPasswordRollbackFailed  = errors.New("password revocation failed and credential rollback failed")
	errCredentialsChanged      = errors.New("administrator credentials changed during login")
)

type LoginRoute struct {
	Config            *config.Config
	Verifier          *oidc.IDTokenVerifier
	SaveAdminPassword func(string) error
	AuthState         *authsetup.Manager
}

// NewVersionRoute creates a new version route
func NewLoginRoute(configData *config.Config, authState ...*authsetup.Manager) *LoginRoute {
	saveAdminPassword := config.SaveAdminPassword

	if configData.SecretBackend == secretBackendKeyring {
		store := security.NewKeyRingStorage(authsetup.KeyringServiceName)
		saveAdminPassword = func(passwordHash string) error {
			return store.SetKeyValue(keyringAdminPassword, passwordHash)
		}
	}

	lr := &LoginRoute{
		Config:            configData,
		SaveAdminPassword: saveAdminPassword,
	}
	if len(authState) > 0 {
		lr.AuthState = authState[0]
	}

	if configData.ClientID != "" {
		ctx := context.Background()

		if configData.TLSSkipVerify {
			transport, _ := http.DefaultTransport.(*http.Transport)
			transport = transport.Clone()
			transport.TLSClientConfig = &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true, //nolint:gosec // operator opted in via auth.tlsSkipVerify to trust self-signed IdP
			}

			ctx = oidc.ClientContext(ctx, &http.Client{Transport: transport})
		}

		provider, err := oidc.NewProvider(ctx, configData.Issuer)
		if err != nil {
			return nil
		}

		lr.Verifier = provider.Verifier(&oidc.Config{
			ClientID: configData.ClientID,
		})
	}

	return lr
}

// ChangePassword updates standalone credentials after verifying the current password.
func (lr LoginRoute) ChangePassword(c *gin.Context) {
	var request dto.PasswordChangeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: "invalid request"})

		return
	}

	if message := validatePasswordChange(request); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: message})

		return
	}

	if _, managed := os.LookupEnv("AUTH_ADMIN_PASSWORD"); managed && lr.Config.SecretBackend != secretBackendKeyring {
		c.JSON(http.StatusConflict, gin.H{errorKey: "password is managed by AUTH_ADMIN_PASSWORD; update the deployment environment and restart Console"})

		return
	}

	authsetup.CredentialCacheMutex.Lock()
	defer authsetup.CredentialCacheMutex.Unlock()

	if lr.SaveAdminPassword == nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: "password storage is unavailable"})

		return
	}

	result, err := lr.persistPasswordChange(request)
	if err != nil {
		message := "could not safely persist password revocation state"
		if errors.Is(err, errPasswordRollbackFailed) {
			message = "password revocation failed and credential rollback failed; administrator recovery is required"
		}

		c.JSON(http.StatusInternalServerError, gin.H{errorKey: message})

		return
	}

	if result.samePassword {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: "new password must differ from current password"})

		return
	}

	if !result.currentPasswordValid {
		c.JSON(http.StatusUnauthorized, gin.H{errorKey: loginFailureError, messageKey: incorrectCredentialsMessage})

		return
	}

	lr.Config.AdminPassword = result.storedPassword

	c.JSON(http.StatusOK, dto.PasswordChangeResponse{Message: "password changed"})
}

type passwordChangeResult struct {
	storedPassword       string
	currentPasswordValid bool
	samePassword         bool
}

func validatePasswordChange(request dto.PasswordChangeRequest) string {
	switch {
	case request.CurrentPassword == "" || request.NewPassword == "" || request.ConfirmPassword == "":
		return "current password, new password, and confirmation are required"
	case request.NewPassword != request.ConfirmPassword:
		return "password confirmation does not match"
	case !secrets.IsAdminPasswordStrong(request.NewPassword):
		return "new password must be at least 8 characters and contain at least 4 distinct characters"
	default:
		return ""
	}
}

func (lr LoginRoute) persistPasswordChange(request dto.PasswordChangeRequest) (passwordChangeResult, error) {
	var result passwordChangeResult

	var previousPassword string

	passwordStored := false

	err := lr.withAuthVersion(func(version uint64) (uint64, error) {
		_, currentPassword, err := lr.currentCredentials()
		if err != nil {
			return version, err
		}

		previousPassword = currentPassword

		result.currentPasswordValid = matchesStoredPassword(lr.Config.SecretBackend, currentPassword, request.CurrentPassword)
		if !result.currentPasswordValid {
			return version, nil
		}

		if request.NewPassword == request.CurrentPassword {
			result.samePassword = true

			return version, nil
		}

		result.storedPassword, err = lr.newStoredPassword(request.NewPassword)
		if err != nil {
			return version, err
		}

		if err := lr.SaveAdminPassword(result.storedPassword); err != nil {
			return version, fmt.Errorf("store new administrator password: %w", err)
		}

		passwordStored = true

		return version + 1, nil
	})
	// Only the auth_version bump (not the preceding keyring/config write) can
	// fail at this point, since passwordStored is already true. The rollback
	// re-enters withAuthVersion and only restores the previous password if the
	// store still holds the value this request wrote, so it can never clobber a
	// concurrent change that completed in between.
	//
	// Revocation does not depend on this rollback or on auth_version alone:
	// every token also carries a tag bound to the stored credential, so a crash
	// between the keyring write and the SQLite commit still invalidates tokens
	// issued against the old password.
	if err != nil && passwordStored {
		if rollbackErr := lr.rollbackStoredPassword(result.storedPassword, previousPassword); rollbackErr != nil {
			return result, errors.Join(errPasswordRollbackFailed, err, rollbackErr)
		}
	}

	return result, err
}

// rollbackStoredPassword restores previousPassword only while the store still
// holds writtenPassword, so a concurrent change that completed in between is
// never overwritten.
func (lr LoginRoute) rollbackStoredPassword(writtenPassword, previousPassword string) error {
	return lr.withAuthVersion(func(version uint64) (uint64, error) {
		_, current, err := lr.currentCredentials()
		if err != nil {
			return version, err
		}

		if current != writtenPassword {
			return version, nil
		}

		return version, lr.SaveAdminPassword(previousPassword)
	})
}

func (lr LoginRoute) newStoredPassword(password string) (string, error) {
	if lr.Config.SecretBackend != secretBackendKeyring {
		return password, nil
	}

	hash, err := secrets.GeneratePBKDF2Hash(password)
	if err != nil {
		return "", fmt.Errorf("hash new administrator password: %w", err)
	}

	return hash, nil
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

	authsetup.CredentialCacheMutex.RLock()

	username, storedPassword, err := lr.currentCredentials()

	authsetup.CredentialCacheMutex.RUnlock()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: "authentication state is unavailable"})

		return
	}

	// PBKDF2 verification (expensive by design) runs here, outside the
	// SQLite-backed auth-state lock, so a flood of failed login attempts
	// can't serialize every other request that needs that lock (token
	// verification, password changes).
	usernameMatches := subtle.ConstantTimeCompare([]byte(username), []byte(creds.Username)) == 1
	if !usernameMatches || !matchesStoredPassword(lr.Config.SecretBackend, storedPassword, creds.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{errorKey: loginFailureError, messageKey: incorrectCredentialsMessage})

		return
	}

	var tokenString string

	var expirationTime time.Time

	// Re-read the stored credential under the auth-state lock so a password
	// change that landed between verification and issuance yields a token
	// bound to the new credential (which the supplied password would then
	// fail against on the next request) rather than the stale one.
	err = lr.withAuthVersion(func(version uint64) (uint64, error) {
		_, lockedPassword, credErr := lr.currentCredentials()
		if credErr != nil {
			return version, credErr
		}

		if lockedPassword != storedPassword {
			return version, errCredentialsChanged
		}

		var tokenErr error

		tokenString, expirationTime, tokenErr = lr.createSessionToken(version, lockedPassword)

		return version, tokenErr
	})

	if errors.Is(err, errCredentialsChanged) {
		c.JSON(http.StatusUnauthorized, gin.H{errorKey: loginFailureError, messageKey: incorrectCredentialsMessage})

		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: "authentication state is unavailable"})

		return
	}

	setSessionCookies(c, tokenString, expirationTime)
	c.JSON(http.StatusOK, gin.H{tokenKey: tokenString})
}

func (lr LoginRoute) createSessionToken(version uint64, storedPassword string) (string, time.Time, error) {
	expirationTime := time.Now().Add(lr.Config.JWTExpiration)
	claims := jwt.MapClaims{
		"exp":          expirationTime.Unix(),
		"auth_version": version,
	}

	// Bind the token to the credential it was issued against. A password
	// change (or keyring recovery, or a crash between the credential write and
	// the auth_version commit) changes the stored value, so every earlier token
	// fails this check even if auth_version did not advance.
	if lr.AuthState != nil {
		claims[credentialTagClaim] = lr.credentialTag(storedPassword)
	}

	// Conditional so the claim set still matches what callers saw before.
	if config.ConsoleConfig.Issuer != "" {
		claims["iss"] = config.ConsoleConfig.Issuer
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(lr.Config.JWTKey))

	return tokenString, expirationTime, err
}

// credentialTag derives a keyed, non-reversible tag from the stored credential
// so the token can be checked against the current credential without
// embedding the hash itself.
func (lr LoginRoute) credentialTag(storedPassword string) string {
	mac := hmac.New(sha256.New, []byte(lr.Config.JWTKey))
	mac.Write([]byte(storedPassword))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (lr LoginRoute) credentialsAccepted(creds dto.Credentials) bool {
	authsetup.CredentialCacheMutex.RLock()
	defer authsetup.CredentialCacheMutex.RUnlock()

	username, storedPassword, err := lr.currentCredentials()

	usernameMatches := subtle.ConstantTimeCompare([]byte(creds.Username), []byte(username)) == 1

	return err == nil && matchesStoredPassword(lr.Config.SecretBackend, storedPassword, creds.Password) &&
		usernameMatches
}

func (lr LoginRoute) currentCredentials() (username, password string, err error) {
	if lr.AuthState != nil {
		return lr.AuthState.CurrentCredentials()
	}

	return lr.Config.AdminUsername, lr.Config.AdminPassword, nil
}

func (lr LoginRoute) withAuthVersion(fn func(uint64) (uint64, error)) error {
	if lr.AuthState != nil {
		return lr.AuthState.WithAuthVersion(fn)
	}

	_, err := fn(0)

	return err
}

func matchesStoredPassword(backend, storedPassword, suppliedPassword string) bool {
	switch backend {
	case secretBackendKeyring:
		return secrets.IsPBKDF2Hash(storedPassword) && secrets.VerifyPBKDF2Hash(storedPassword, suppliedPassword)
	case "config":
		return storedPassword != "" && subtle.ConstantTimeCompare([]byte(suppliedPassword), []byte(storedPassword)) == 1
	default:
		return false
	}
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

	if lr.AuthState == nil {
		return true
	}

	version, ok := (*claims)["auth_version"].(float64)
	if !ok || version < 0 {
		return false
	}

	currentVersion, err := lr.AuthState.AuthVersion()
	if err != nil || uint64(version) != currentVersion {
		return false
	}

	tag, ok := (*claims)[credentialTagClaim].(string)
	if !ok || tag == "" {
		return false
	}

	_, storedPassword, err := lr.currentCredentials()
	if err != nil || storedPassword == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(tag), []byte(lr.credentialTag(storedPassword))) == 1
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
