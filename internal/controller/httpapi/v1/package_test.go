package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/packaging"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type stubPackaging struct {
	releases []dto.RPCRelease
	zip      []byte
	err      error
	tenantID string
}

func (s *stubPackaging) ListVersions(_ context.Context) ([]dto.RPCRelease, error) {
	return s.releases, s.err
}

func (s *stubPackaging) BuildPackage(_ context.Context, _ dto.PackageRequest, tenantID string) (io.Reader, string, error) {
	s.tenantID = tenantID

	if s.err != nil {
		return nil, "", s.err
	}

	return bytes.NewReader(s.zip), "rpc-activate-linux-x86_64.zip", nil
}

func newPackageEngine(stub *stubPackaging) *gin.Engine {
	log := logger.New("error")
	engine := gin.New()

	// Mirror router.go, which resolves the tenant on the protected group before
	// the package routes are mounted.
	group := engine.Group("/api")
	group.Use(middleware.ResolveTenant(log))

	NewPackageRoutes(group, stub, log)

	return engine
}

func TestPackageRoutes(t *testing.T) {
	t.Parallel()

	t.Run("GET rpc-versions returns 200 with releases", func(t *testing.T) {
		t.Parallel()

		releases := []dto.RPCRelease{
			{Version: "v1.2.3", Assets: []dto.RPCAsset{{OS: "linux", Arch: "x86_64"}}},
		}
		engine := newPackageEngine(&stubPackaging{releases: releases})

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/package/rpc-versions", http.NoBody)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		wantJSON, err := json.Marshal(releases)
		require.NoError(t, err)
		require.Equal(t, string(wantJSON), w.Body.String())
	})

	t.Run("POST package with invalid body returns 400", func(t *testing.T) {
		t.Parallel()

		// Malformed JSON triggers a JSON-decode error from ShouldBindJSON,
		// which the handler wraps as a NotValidError → 400 Bad Request.
		// (gin.DisableBindValidation is set in init() so struct-tag validation
		// is not active in tests; a JSON-syntax error is the reliable way to
		// exercise the 400 path.)
		body := `{not valid json`
		engine := newPackageEngine(&stubPackaging{})

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/package", bytes.NewBufferString(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("POST package with valid body returns 200 zip", func(t *testing.T) {
		t.Parallel()

		zipData := []byte("PK\x03\x04fake-zip-content")
		engine := newPackageEngine(&stubPackaging{zip: zipData})

		reqBody := dto.PackageRequest{
			Command: "activate",
			Version: "v1.2.3",
			OS:      "linux",
			Arch:    "x86_64",
			Auth:    dto.PackageAuth{Mode: "token"},
		}

		bodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/package", bytes.NewBuffer(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Header().Get("Content-Type"), "application/zip")
		require.Equal(t, zipData, w.Body.Bytes())
	})

	t.Run("POST package returns 400 when the token lifetime exceeds the server maximum", func(t *testing.T) {
		t.Parallel()

		engine := newPackageEngine(&stubPackaging{err: packaging.ErrTokenTTLTooLong})

		reqBody := dto.PackageRequest{
			Command:  "activate",
			Version:  "v1.2.3",
			OS:       "linux",
			Arch:     "x86_64",
			Auth:     dto.PackageAuth{Mode: "token"},
			TokenTTL: "24h",
		}

		bodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/package", bytes.NewBuffer(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GET rpc-versions returns 5xx when ListVersions errors", func(t *testing.T) {
		t.Parallel()

		stubErr := errors.New("upstream unavailable")
		engine := newPackageEngine(&stubPackaging{err: stubErr})

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/package/rpc-versions", http.NoBody)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.GreaterOrEqual(t, w.Code, http.StatusInternalServerError)
	})

	t.Run("POST package returns 5xx when BuildPackage errors", func(t *testing.T) {
		t.Parallel()

		stubErr := errors.New("build failure")
		engine := newPackageEngine(&stubPackaging{err: stubErr})

		reqBody := dto.PackageRequest{
			Command: "activate",
			Version: "v1.2.3",
			OS:      "linux",
			Arch:    "x86_64",
			Auth:    dto.PackageAuth{Mode: "token"},
		}

		bodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/package", bytes.NewBuffer(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.GreaterOrEqual(t, w.Code, http.StatusInternalServerError)
	})
}

// postPackage issues a build request against a stub and returns the recorder.
func postPackage(t *testing.T, stub *stubPackaging, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	engine := newPackageEngine(stub)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/package", bytes.NewBufferString(validPackageBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	return w
}

const validPackageBody = `{"command":"activate","version":"v3.0.1","os":"linux","arch":"x86_64",` +
	`"auth":{"mode":"userpass","username":"u","password":"p"},"profile":"p1"}`

// The resolved tenant must reach the use case; without it a package built by one
// tenant carries no tenant scope and resolves against the default tenant's data.
func TestPackageBuildPropagatesTenant(t *testing.T) {
	t.Parallel()

	stub := &stubPackaging{zip: []byte("zip-bytes")}

	w := postPackage(t, stub, map[string]string{"x-tenant-id": "acme-corp"})

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "acme-corp", stub.tenantID)
}

func TestPackageBuildDefaultTenantWhenHeaderAbsent(t *testing.T) {
	t.Parallel()

	stub := &stubPackaging{zip: []byte("zip-bytes")}

	w := postPackage(t, stub, nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, stub.tenantID)
}

// Bad input must not read as a server fault.
func TestPackageBuildMapsUserErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"unknown asset is not found", packaging.ErrAssetNotFound, http.StatusNotFound},
		{"traversal version is a bad request", packaging.ErrUnsafeVersion, http.StatusBadRequest},
		{"token mode under OIDC is a bad request", packaging.ErrTokenModeUnsupported, http.StatusBadRequest},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := postPackage(t, &stubPackaging{err: tt.err}, nil)

			require.Equal(t, tt.wantCode, w.Code)
		})
	}
}

// Wrapped errors must map the same way — BuildPackage wraps with %w.
func TestPackageBuildMapsWrappedAssetNotFound(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("resolve asset: %w", packaging.ErrAssetNotFound)

	w := postPackage(t, &stubPackaging{err: wrapped}, nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}
