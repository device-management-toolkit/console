package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/secrets"
)

const (
	testPasswordURL  = "/api/v1/authorize/password"
	testNewAdminPass = "New-Secret-123!"
)

// memoryKeyring is an in-memory security.Storager standing in for the OS keyring.
type memoryKeyring map[string]string

func (k memoryKeyring) GetKeyValue(key string) (string, error) {
	value, ok := k[key]
	if !ok {
		return "", security.ErrKeyNotFound
	}

	return value, nil
}

func (k memoryKeyring) SetKeyValue(key, value string) error {
	k[key] = value

	return nil
}

func (k memoryKeyring) DeleteKeyValue(key string) error {
	delete(k, key)

	return nil
}

// newKeyringTestEngine is newAuthTestEngine with a keyring and the password route.
func newKeyringTestEngine(t *testing.T) (*gin.Engine, memoryKeyring) {
	t.Helper()

	cfg := cookieAuthTestConfig(t)
	keyring := memoryKeyring{
		secrets.KeyringAdminUsernameKey: cfg.AdminUsername,
		secrets.KeyringAdminPasswordKey: cfg.AdminPassword,
	}

	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	config.ConsoleConfig = cfg

	route := LoginRoute{Config: cfg, Keyring: keyring}

	engine := gin.New()
	engine.POST(testAuthorizeURL, route.Login)
	engine.GET(testProtectedURL, route.JWTAuthMiddleware(), func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.POST(testPasswordURL, route.JWTAuthMiddleware(), route.ChangePassword)

	return engine, keyring
}

func postJSON(t *testing.T, engine *gin.Engine, url, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set(authorizationHeader, bearerPrefix+token)
	}

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	return w
}

func changePassword(t *testing.T, engine *gin.Engine, token, current, newPassword, confirm string) *httptest.ResponseRecorder {
	t.Helper()

	return postJSON(t, engine, testPasswordURL, token, map[string]string{
		"currentPassword": current, "newPassword": newPassword, "confirmPassword": confirm,
	})
}

func loginStatus(t *testing.T, engine *gin.Engine, password string) int {
	t.Helper()

	return postJSON(t, engine, testAuthorizeURL, "", map[string]string{"username": testAdminUser, "password": password}).Code
}

//nolint:paralleltest // shared global config.ConsoleConfig
func TestChangePasswordStoresHashAndRevokesSessions(t *testing.T) {
	engine, keyring := newKeyringTestEngine(t)

	token, _ := login(t, engine)
	require.Equal(t, http.StatusOK, get(t, engine, withBearer(token)).Code)

	w := changePassword(t, engine, token, testAdminPass, testNewAdminPass, testNewAdminPass)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	stored := keyring[secrets.KeyringAdminPasswordKey]
	require.True(t, secrets.IsPBKDF2Hash(stored), "keyring must hold a hash, not the password")
	require.True(t, secrets.VerifyPBKDF2Hash(stored, testNewAdminPass))

	require.Equal(t, http.StatusUnauthorized, get(t, engine, withBearer(token)).Code,
		"tokens issued before the change, including the caller's, must be rejected")
	require.Equal(t, http.StatusUnauthorized, loginStatus(t, engine, testAdminPass))
	require.Equal(t, http.StatusOK, loginStatus(t, engine, testNewAdminPass))
}

//nolint:paralleltest // shared global config.ConsoleConfig
func TestChangePasswordRejectsInvalidRequests(t *testing.T) {
	engine, keyring := newKeyringTestEngine(t)
	token, _ := login(t, engine)
	original := keyring[secrets.KeyringAdminPasswordKey]

	tests := []struct {
		name                   string
		current, next, confirm string
		want                   int
	}{
		{"missing fields", "", testNewAdminPass, testNewAdminPass, http.StatusBadRequest},
		{"confirmation mismatch", testAdminPass, testNewAdminPass, testNewAdminPass + "x", http.StatusBadRequest},
		{"weak password", testAdminPass, "weakpass", "weakpass", http.StatusBadRequest},
		{"wrong current password", "Wrong-Pass-1!", testNewAdminPass, testNewAdminPass, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := changePassword(t, engine, token, tt.current, tt.next, tt.confirm)
			require.Equal(t, tt.want, w.Code, w.Body.String())
			require.Equal(t, original, keyring[secrets.KeyringAdminPasswordKey], "stored hash must be unchanged")
		})
	}

	require.Equal(t, http.StatusUnauthorized, changePassword(t, engine, "", testAdminPass, testNewAdminPass, testNewAdminPass).Code,
		"an unauthenticated request must be rejected")
}

//nolint:paralleltest // shared global config.ConsoleConfig
func TestChangePasswordRejectsReuse(t *testing.T) {
	engine, keyring := newKeyringTestEngine(t)

	keyring[secrets.KeyringAdminPasswordKey] = mustHash(t, testNewAdminPass)
	token := loginToken(t, engine, testNewAdminPass)

	w := changePassword(t, engine, token, testNewAdminPass, testNewAdminPass, testNewAdminPass)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "must differ")
}

//nolint:paralleltest // shared global config.ConsoleConfig
func TestOtherBackendsKeepOriginalBehavior(t *testing.T) {
	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	for _, cfg := range []*config.Config{
		{Auth: config.Auth{SecretBackend: "config"}},
		{Auth: config.Auth{SecretBackend: "keyring", Disabled: true}},
	} {
		config.ConsoleConfig = cfg
		require.Nil(t, NewLoginRoute(cfg).Keyring, "password change must not be enabled for %+v", cfg.Auth)
	}

	cfg := cookieAuthTestConfig(t)
	cfg.SecretBackend = "config"
	cfg.AdminPassword = testAdminPass
	engine := newAuthTestEngine(t, cfg)

	token, _ := login(t, engine)

	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(token, claims)
	require.NoError(t, err)
	require.Equal(t, []string{"exp"}, mapKeys(claims), "config-backend tokens keep their original claims")
}

func mustHash(t *testing.T, password string) string {
	t.Helper()

	hash, err := secrets.GeneratePBKDF2Hash(password)
	require.NoError(t, err)

	return hash
}

func loginToken(t *testing.T, engine *gin.Engine, password string) string {
	t.Helper()

	w := postJSON(t, engine, testAuthorizeURL, "", map[string]string{"username": testAdminUser, "password": password})
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]string

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	return body[tokenKey]
}

func mapKeys(m jwt.MapClaims) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	return keys
}
