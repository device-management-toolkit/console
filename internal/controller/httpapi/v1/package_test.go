package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type stubPackaging struct {
	releases []dto.RPCRelease
	zip      []byte
	err      error
}

func (s *stubPackaging) ListVersions(_ context.Context) ([]dto.RPCRelease, error) {
	return s.releases, s.err
}

func (s *stubPackaging) BuildPackage(_ context.Context, _ dto.PackageRequest) (io.Reader, string, error) {
	if s.err != nil {
		return nil, "", s.err
	}

	return bytes.NewReader(s.zip), "rpc-activate-linux-x86_64.zip", nil
}

func newPackageEngine(stub *stubPackaging) *gin.Engine {
	log := logger.New("error")
	engine := gin.New()

	NewPackageRoutes(engine.Group("/api"), stub, log)

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
