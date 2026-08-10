package secrets

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/device-management-toolkit/console/config"
)

const testVaultPath = "secret/data/console"

type vaultHandlerFunc func(method, path string, body []byte) (int, string)

func newTestVaultClient(t *testing.T, handler vaultHandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		status, resp := handler(r.Method, r.URL.Path, body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(resp))
	}))
	t.Cleanup(server.Close)

	cfg := &config.Secrets{Address: server.URL, Token: "test-token", Path: testVaultPath}
	client, err := NewClient(cfg)
	assert.NoError(t, err)

	return client
}

func TestValidatePathKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		key       string
		wantError bool
	}{
		{name: "valid nested", key: "certificates/root", wantError: false},
		{name: "valid simple", key: "encryption-key", wantError: false},
		{name: "valid underscore and dot", key: "certs/root_v1.pem", wantError: false},
		{name: "empty", key: "", wantError: true},
		{name: "absolute-like", key: "/certificates/root", wantError: true},
		{name: "parent traversal", key: "../other/path", wantError: true},
		{name: "inline traversal", key: "certs/../../other", wantError: true},
		{name: "dot segment", key: "./certs/root", wantError: true},
		{name: "double slash", key: "certs//root", wantError: true},
		{name: "backslash separator", key: "certs\\root", wantError: true},
		{name: "query char", key: "certs/root?x=1", wantError: true},
		{name: "fragment char", key: "certs/root#frag", wantError: true},
		{name: "percent encoded slash", key: "certs%2froot", wantError: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validatePathKey(tc.key)
			if tc.wantError {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestValidatePathKey_TraversalBranch(t *testing.T) {
	t.Parallel()

	err := validatePathKey("..")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)
	assert.Contains(t, err.Error(), "traversal not allowed")
}

func TestValidatePathKey_NonNormalizedBranch(t *testing.T) {
	t.Parallel()

	err := validatePathKey("a//b")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)
	assert.Contains(t, err.Error(), "non-normalized path not allowed")
}

func TestBuildScopedSecretPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		basePath  string
		key       string
		wantPath  string
		wantError bool
	}{
		{
			name:      "valid nested path",
			basePath:  "secret/data/console",
			key:       "certificates/root",
			wantPath:  "secret/data/console/certificates/root",
			wantError: false,
		},
		{
			name:      "base path with trailing slash",
			basePath:  "secret/data/console/",
			key:       "keys",
			wantPath:  "secret/data/console/keys",
			wantError: false,
		},
		{
			name:      "reject traversal",
			basePath:  "secret/data/console",
			key:       "../other/path",
			wantError: true,
		},
		{
			name:      "reject absolute-like",
			basePath:  "secret/data/console",
			key:       "/certificates/root",
			wantError: true,
		},
		{
			name:      "reject unsupported chars",
			basePath:  "secret/data/console",
			key:       "certs/root?x=1",
			wantError: true,
		},
		{
			name:      "reject when base namespace is empty",
			basePath:  "",
			key:       "keys",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotPath, err := buildScopedSecretPath(tc.basePath, tc.key)
			if tc.wantError {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.wantPath, gotPath)
		})
	}
}

func TestPathBasedMethods_RejectInvalidPathKey(t *testing.T) {
	t.Parallel()

	client := &Client{path: DefaultSecretPath}
	invalidPathKey := "../other/path"

	_, err := client.GetKeyValue(invalidPathKey)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)

	err = client.SetKeyValue(invalidPathKey, "value")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)

	err = client.DeleteKeyValue(invalidPathKey)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)

	_, err = client.GetObject(invalidPathKey)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)

	err = client.SetObject(invalidPathKey, map[string]string{"k": "v"})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)
}

func TestGetKeyValue_PathBasedSuccess(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(method, reqPath string, _ []byte) (int, string) {
		assert.Equal(t, http.MethodGet, method)
		assert.Equal(t, "/v1/secret/data/console/certs/root", reqPath)

		return http.StatusOK, `{"data":{"data":{"value":"abc123"}}}`
	})

	value, err := client.GetKeyValue("certs/root")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", value)
}

func TestGetKeyValue_FieldBasedSuccess(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(method, reqPath string, _ []byte) (int, string) {
		assert.Equal(t, http.MethodGet, method)
		assert.Equal(t, "/v1/secret/data/console/keys", reqPath)

		return http.StatusOK, `{"data":{"data":{"device-password":"secret"}}}`
	})

	value, err := client.GetKeyValue("device-password")
	assert.NoError(t, err)
	assert.Equal(t, "secret", value)
}

func TestGetKeyValue_FieldBasedInvalidBasePath(t *testing.T) {
	t.Parallel()

	// Empty base path causes buildScopedSecretPath("", "keys") to fail,
	// which should be returned from GetKeyValue's field-based branch.
	client := &Client{path: ""}

	_, err := client.GetKeyValue("device-password")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)
	assert.Contains(t, err.Error(), "path escapes base namespace")
}

func TestGetKeyValue_ReadError(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(_, _ string, _ []byte) (int, string) {
		return http.StatusInternalServerError, `{"errors":["backend down"]}`
	})

	_, err := client.GetKeyValue("device-password")
	assert.Error(t, err)
}

func TestGetKeyValue_SecretNotFound(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(_, _ string, _ []byte) (int, string) {
		return http.StatusNotFound, `{"errors":[]}`
	})

	_, err := client.GetKeyValue("device-password")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrSecretNotFound), "expected ErrSecretNotFound, got %v", err)
}

func TestGetKeyValue_UnexpectedDataFormat(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(_, _ string, _ []byte) (int, string) {
		return http.StatusOK, `{"data":{"not_data":{"device-password":"secret"}}}`
	})

	_, err := client.GetKeyValue("device-password")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnexpectedDataFormat), "expected ErrUnexpectedDataFormat, got %v", err)
}

func TestGetKeyValue_KeyNotFound(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(_, _ string, _ []byte) (int, string) {
		return http.StatusOK, `{"data":{"data":{"some-other-key":"secret"}}}`
	})

	_, err := client.GetKeyValue("device-password")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrKeyNotFound), "expected ErrKeyNotFound, got %v", err)
}

func TestGetKeyValue_ValueNotString(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(_, _ string, _ []byte) (int, string) {
		return http.StatusOK, `{"data":{"data":{"device-password":42}}}`
	})

	_, err := client.GetKeyValue("device-password")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrValueNotString), "expected ErrValueNotString, got %v", err)
}

func TestSetKeyValue_PathBasedSuccess(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(method, reqPath string, body []byte) (int, string) {
		assert.Equal(t, http.MethodPut, method)
		assert.Equal(t, "/v1/secret/data/console/certs/root", reqPath)
		assert.Contains(t, string(body), `"value":"abc123"`)

		return http.StatusOK, `{"data":{}}`
	})

	err := client.SetKeyValue("certs/root", "abc123")
	assert.NoError(t, err)
}

func TestSetKeyValue_FieldBasedSuccessAndMerge(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(method, reqPath string, body []byte) (int, string) {
		switch method {
		case http.MethodGet:
			assert.Equal(t, "/v1/secret/data/console/keys", reqPath)

			return http.StatusOK, `{"data":{"data":{"existing":"keep"}}}`
		case http.MethodPut:
			assert.Equal(t, "/v1/secret/data/console/keys", reqPath)

			var payload map[string]map[string]string

			err := json.Unmarshal(body, &payload)
			assert.NoError(t, err)
			assert.Equal(t, "keep", payload["data"]["existing"])
			assert.Equal(t, "new-value", payload["data"]["new-key"])

			return http.StatusOK, `{"data":{}}`
		default:
			return http.StatusMethodNotAllowed, `{"errors":["unexpected method"]}`
		}
	})

	err := client.SetKeyValue("new-key", "new-value")
	assert.NoError(t, err)
}

func TestSetKeyValue_FieldBasedReadErrorStillWrites(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(method, reqPath string, body []byte) (int, string) {
		switch method {
		case http.MethodGet:
			return http.StatusInternalServerError, `{"errors":["backend down"]}`
		case http.MethodPut:
			assert.Equal(t, "/v1/secret/data/console/keys", reqPath)
			assert.Contains(t, string(body), `"new-key":"new-value"`)
			assert.False(t, strings.Contains(string(body), "existing"))

			return http.StatusOK, `{"data":{}}`
		default:
			return http.StatusMethodNotAllowed, `{"errors":["unexpected method"]}`
		}
	})

	err := client.SetKeyValue("new-key", "new-value")
	assert.NoError(t, err)
}

func TestSetKeyValue_FieldBasedInvalidBasePath(t *testing.T) {
	t.Parallel()

	// Empty base path causes buildScopedSecretPath("", "keys") to fail,
	// which should be returned from setFieldKeyValue via SetKeyValue.
	client := &Client{path: ""}

	err := client.SetKeyValue("new-key", "new-value")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)
	assert.Contains(t, err.Error(), "path escapes base namespace")
}

func TestDeleteKeyValue_PathBasedSuccess(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(method, reqPath string, _ []byte) (int, string) {
		assert.Equal(t, http.MethodDelete, method)
		assert.Equal(t, "/v1/secret/metadata/console/certs/root", reqPath)

		return http.StatusNoContent, ``
	})

	err := client.DeleteKeyValue("certs/root")
	assert.NoError(t, err)
}

func TestDeleteKeyValue_FieldBasedReadError(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(_, _ string, _ []byte) (int, string) {
		return http.StatusInternalServerError, `{"errors":["backend down"]}`
	})

	err := client.DeleteKeyValue("device-password")
	assert.Error(t, err)
}

func TestDeleteKeyValue_FieldBasedInvalidBasePath(t *testing.T) {
	t.Parallel()

	// Empty base path causes buildScopedSecretPath("", "keys") to fail,
	// which should be returned from DeleteKeyValue's field-based branch.
	client := &Client{path: ""}

	err := client.DeleteKeyValue("device-password")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)
	assert.Contains(t, err.Error(), "path escapes base namespace")
}

func TestDeleteKeyValue_FieldBasedSecretNotFound(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(_, _ string, _ []byte) (int, string) {
		return http.StatusNotFound, `{"errors":[]}`
	})

	err := client.DeleteKeyValue("device-password")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrSecretNotFound), "expected ErrSecretNotFound, got %v", err)
}

func TestDeleteKeyValue_FieldBasedUnexpectedDataFormat(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(method, _ string, _ []byte) (int, string) {
		switch method {
		case http.MethodGet:
			return http.StatusOK, `{"data":{"not_data":{"k":"v"}}}`
		case http.MethodPut:
			return http.StatusOK, `{"data":{}}`
		default:
			return http.StatusMethodNotAllowed, `{"errors":["unexpected method"]}`
		}
	})

	err := client.DeleteKeyValue("device-password")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnexpectedDataFormat), "expected ErrUnexpectedDataFormat, got %v", err)
}

func TestDeleteKeyValue_FieldBasedSuccess(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(method, _ string, body []byte) (int, string) {
		switch method {
		case http.MethodGet:
			return http.StatusOK, `{"data":{"data":{"device-password":"secret","other":"keep"}}}`
		case http.MethodPut:
			assert.Contains(t, string(body), `"other":"keep"`)
			assert.False(t, strings.Contains(string(body), "device-password"))

			return http.StatusOK, `{"data":{}}`
		default:
			return http.StatusMethodNotAllowed, `{"errors":["unexpected method"]}`
		}
	})

	err := client.DeleteKeyValue("device-password")
	assert.NoError(t, err)
}

func TestGetObject_Success(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(method, reqPath string, _ []byte) (int, string) {
		assert.Equal(t, http.MethodGet, method)
		assert.Equal(t, "/v1/secret/data/console/certs/root", reqPath)

		return http.StatusOK, `{"data":{"data":{"cert":"pem-data","version":7}}}`
	})

	obj, err := client.GetObject("certs/root")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"cert": "pem-data"}, obj)
}

func TestGetObject_SecretNotFound(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(_, _ string, _ []byte) (int, string) {
		return http.StatusNotFound, `{"errors":[]}`
	})

	_, err := client.GetObject("certs/root")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrSecretNotFound), "expected ErrSecretNotFound, got %v", err)
}

func TestGetObject_UnexpectedDataFormat(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(_, _ string, _ []byte) (int, string) {
		return http.StatusOK, `{"data":{"not_data":{"cert":"pem"}}}`
	})

	_, err := client.GetObject("certs/root")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnexpectedDataFormat), "expected ErrUnexpectedDataFormat, got %v", err)
}

func TestSetObject_Success(t *testing.T) {
	t.Parallel()

	client := newTestVaultClient(t, func(method, reqPath string, body []byte) (int, string) {
		assert.Equal(t, http.MethodPut, method)
		assert.Equal(t, "/v1/secret/data/console/certs/root", reqPath)
		assert.Contains(t, string(body), `"cert":"pem-data"`)
		assert.Contains(t, string(body), `"key":"key-data"`)

		return http.StatusOK, `{"data":{}}`
	})

	err := client.SetObject("certs/root", map[string]string{"cert": "pem-data", "key": "key-data"})
	assert.NoError(t, err)
}

func TestGetObject_RequiresPathKey(t *testing.T) {
	t.Parallel()

	client := &Client{path: DefaultSecretPath}

	_, err := client.GetObject("certs")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)
	assert.Contains(t, err.Error(), "must contain '/'")
}

func TestSetObject_RequiresPathKey(t *testing.T) {
	t.Parallel()

	client := &Client{path: DefaultSecretPath}

	err := client.SetObject("certs", map[string]string{"cert": "pem-data"})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPathKey), "expected ErrInvalidPathKey, got %v", err)
	assert.Contains(t, err.Error(), "must contain '/'")
}
