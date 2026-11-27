package authsetup

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/pkg/secrets"
)

func serve(handler http.Handler, method, path, remoteAddr, origin, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = remoteAddr

	if origin != "" {
		req.Header.Set("Origin", origin)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	return w
}

func TestHandlerRejectsNonLoopbackRequests(t *testing.T) {
	t.Parallel()

	manager, _ := newTestManager(t)
	handler := NewHandler(manager)

	for _, path := range []string{setupStatusPath, SetupRedirectPath} {
		w := serve(handler, http.MethodGet, path, "192.0.2.10:5000", "", "")
		require.Equal(t, http.StatusForbidden, w.Code, path)
		require.Contains(t, w.Body.String(), codeLocalOnly)
	}

	w := serve(handler, http.MethodGet, setupStatusPath, "127.0.0.1:5000", "https://evil.example", "")
	require.Equal(t, http.StatusForbidden, w.Code, "non-loopback Origin must be rejected")
}

func TestHandlerRedirectAndSetup(t *testing.T) {
	t.Parallel()

	manager, _ := newTestManager(t)
	handler := NewHandler(manager)
	token := manager.SetupToken()

	w := serve(handler, http.MethodGet, SetupRedirectPath, "127.0.0.1:5000", "", "")
	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, setupPagePath+"#"+token, w.Header().Get("Location"))

	body := `{"token":"` + token + `","username":"admin","password":"` + testPassword + `","confirmPassword":"` + testPassword + `"}`
	require.Equal(t, http.StatusCreated, serve(handler, http.MethodPost, setupPath, "127.0.0.1:5000", "", body).Code)
	require.Equal(t, http.StatusConflict, serve(handler, http.MethodPost, setupPath, "127.0.0.1:5000", "", body).Code)

	w = serve(handler, http.MethodGet, SetupRedirectPath, "[::1]:5000", "", "")
	require.Equal(t, setupPagePath, w.Header().Get("Location"), "no token after setup")
}

func TestHandlerMethodsAndErrors(t *testing.T) {
	t.Parallel()

	manager, store := newTestManager(t)
	handler := NewHandler(manager)
	token := manager.SetupToken()

	const local = "127.0.0.1:5000"

	setupBody := func(tok, password string) string {
		return `{"token":"` + tok + `","username":"admin","password":"` + password + `","confirmPassword":"` + password + `"}`
	}

	tests := []struct {
		name, method, path, body string
		want                     int
		code                     string
	}{
		{"status preflight", http.MethodOptions, setupStatusPath, "", http.StatusNoContent, ""},
		{"status wrong method", http.MethodPost, setupStatusPath, "", http.StatusMethodNotAllowed, ""},
		{"setup preflight", http.MethodOptions, setupPath, "", http.StatusNoContent, ""},
		{"setup wrong method", http.MethodGet, setupPath, "", http.StatusMethodNotAllowed, ""},
		{"redirect wrong method", http.MethodPost, SetupRedirectPath, "", http.StatusMethodNotAllowed, ""},
		{"malformed JSON", http.MethodPost, setupPath, "{", http.StatusBadRequest, "INVALID_REQUEST"},
		{"weak password", http.MethodPost, setupPath, setupBody(token, "weak"), http.StatusBadRequest, "INVALID_CREDENTIALS"},
		{"wrong token", http.MethodPost, setupPath, setupBody("nope", testPassword), http.StatusUnauthorized, "INVALID_SETUP_TOKEN"},
	}

	for _, tt := range tests {
		w := serve(handler, tt.method, tt.path, local, "", tt.body)
		require.Equal(t, tt.want, w.Code, tt.name)
		require.Contains(t, w.Body.String(), tt.code, tt.name)
	}

	w := serve(handler, http.MethodGet, setupStatusPath, local, "https://localhost:4200", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "https://localhost:4200", w.Header().Get("Access-Control-Allow-Origin"), "loopback origins get CORS headers")

	store.setErr[secrets.KeyringAdminUsernameKey] = errors.New("keyring locked")
	w = serve(handler, http.MethodPost, setupPath, local, "", setupBody(token, testPassword))
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "SETUP_FAILED")

	store.getErr = errors.New("keyring locked")
	w = serve(handler, http.MethodGet, setupStatusPath, local, "", "")
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "SETUP_STATUS_FAILED")
}
