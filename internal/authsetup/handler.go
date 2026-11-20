package authsetup

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const (
	setupStatusPath   = "/api/v1/auth/setup/status"
	setupPath         = "/api/v1/auth/setup"
	setupRedirectPath = "/setup/redirect"
	maxSetupBody      = 16 * 1024
	responseCodeKey   = "code"
)

type setupRequest struct {
	Token           string `json:"token"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

func NewHandler(manager *Manager) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(setupStatusPath, handleSetupStatus(manager))
	mux.HandleFunc(setupPath, handleSetup(manager))
	mux.HandleFunc(setupRedirectPath, handleSetupRedirect(manager))

	return mux
}

func handleSetupStatus(manager *Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !allowLoopbackRequest(w, r) {
			return
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		setupRequired, err := manager.SetupRequired()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{responseCodeKey: "SETUP_STATUS_FAILED"})

			return
		}

		writeJSON(w, http.StatusOK, map[string]bool{"setupRequired": setupRequired})
	}
}

func handleSetup(manager *Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !allowLoopbackRequest(w, r) {
			return
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxSetupBody)

		var request setupRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{responseCodeKey: "INVALID_REQUEST"})

			return
		}

		err := manager.Bootstrap(request.Token, request.Username, request.Password, request.ConfirmPassword)
		writeSetupResult(w, err)
	}
}

// handleSetupRedirect lets launchBrowser open a token-free URL instead of
// passing the setup token as a CLI argument, where it would be visible to
// any local user via the process list. The token only appears in the
// Location header of this loopback-only response and the resulting
// browser address bar, matching the exposure the setup page already has.
func handleSetupRedirect(manager *Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !allowLoopbackRequest(w, r) {
			return
		}

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		http.Redirect(w, r, "/setup#"+url.QueryEscape(manager.SetupToken()), http.StatusFound)
	}
}

func writeSetupResult(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAlreadyConfigured):
		writeJSON(w, http.StatusConflict, map[string]string{responseCodeKey: "SETUP_ALREADY_COMPLETED"})
	case errors.Is(err, ErrInvalidSetupToken):
		writeJSON(w, http.StatusUnauthorized, map[string]string{responseCodeKey: "INVALID_SETUP_TOKEN"})
	case errors.Is(err, ErrInvalidCredentials):
		writeJSON(w, http.StatusBadRequest, map[string]string{responseCodeKey: "INVALID_CREDENTIALS"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]string{responseCodeKey: "SETUP_FAILED"})
	default:
		writeJSON(w, http.StatusCreated, map[string]string{"status": "configured"})
	}
}

func allowLoopbackRequest(w http.ResponseWriter, r *http.Request) bool {
	if !remoteIsLoopback(r.RemoteAddr) {
		writeJSON(w, http.StatusForbidden, map[string]string{responseCodeKey: "SETUP_LOCAL_ONLY"})

		return false
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil || !isLoopbackHost(parsed.Hostname()) {
		writeJSON(w, http.StatusForbidden, map[string]string{responseCodeKey: "SETUP_LOCAL_ONLY"})

		return false
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	return true
}

func remoteIsLoopback(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)

	return err == nil && isLoopbackHost(host)
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(strings.Trim(host, "[]"))

	return ip != nil && ip.IsLoopback()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
