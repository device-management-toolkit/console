package v1

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"maps"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
)

const (
	testOIDCClientID = "spa-client"
	testOIDCKeyID    = "test-key"
)

// oidcSigningServer extends oidcDiscoveryServer with a JWKS for one RSA key,
// and returns a function that signs claims with that key as the issuer.
func oidcSigningServer(t *testing.T) (issuer string, sign func(jwt.MapClaims) string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	srv := oidcDiscoveryServer(t)

	mux, ok := srv.Config.Handler.(*http.ServeMux)
	require.True(t, ok)

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": testOIDCKeyID,
				"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
			}},
		})
	})

	sign = func(claims jwt.MapClaims) string {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = testOIDCKeyID

		signed, signErr := token.SignedString(key)
		require.NoError(t, signErr)

		return signed
	}

	return srv.URL, sign
}

// newOIDCTestEngine installs an OAuth-mode config as the singleton and wires
// one protected route behind the JWT middleware.
func newOIDCTestEngine(t *testing.T, issuer string, audiences []string) *gin.Engine {
	t.Helper()

	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	cfg := &config.Config{}
	cfg.ClientID = testOIDCClientID
	cfg.Issuer = issuer
	cfg.Audiences = audiences
	cfg.TLSSkipVerify = true

	config.ConsoleConfig = cfg

	route := NewLoginRoute(cfg)
	require.NotNil(t, route, "expected OIDC discovery to succeed")

	engine := gin.New()
	engine.GET(testProtectedURL, route.JWTAuthMiddleware(), func(c *gin.Context) { c.Status(http.StatusOK) })

	return engine
}

// TestOIDCAudiences pins which access tokens Console accepts in OAuth mode:
// aud must name an allowed audience, and only a token without aud may fall
// back to its client_id claim (Cognito access tokens carry no aud).
//
//nolint:paralleltest // shared global config.ConsoleConfig
func TestOIDCAudiences(t *testing.T) {
	issuer, sign := oidcSigningServer(t)

	claims := func(extra jwt.MapClaims) jwt.MapClaims {
		c := jwt.MapClaims{
			"iss": issuer,
			"sub": "user-1",
			"iat": time.Now().Unix(),
			"exp": time.Now().Add(time.Hour).Unix(),
		}
		maps.Copy(c, extra)

		return c
	}

	tests := []struct {
		name      string
		audiences []string
		claims    jwt.MapClaims
		want      int
	}{
		{"aud is the client ID", nil, claims(jwt.MapClaims{"aud": testOIDCClientID}), http.StatusOK},
		{"aud list contains the client ID", nil, claims(jwt.MapClaims{"aud": []string{"other", testOIDCClientID}}), http.StatusOK},
		{"aud is a configured audience", []string{"https://api.example.com"}, claims(jwt.MapClaims{"aud": "https://api.example.com"}), http.StatusOK},
		{"configured audience is trimmed", []string{" https://api.example.com "}, claims(jwt.MapClaims{"aud": "https://api.example.com"}), http.StatusOK},
		{"aud is not allowed", []string{"https://api.example.com"}, claims(jwt.MapClaims{"aud": "https://other.example.com"}), http.StatusUnauthorized},
		{"no aud, client_id is the client ID", nil, claims(jwt.MapClaims{"client_id": testOIDCClientID, "token_use": "access"}), http.StatusOK},
		{"no aud, client_id is a configured audience", []string{"m2m-client"}, claims(jwt.MapClaims{"client_id": "m2m-client"}), http.StatusOK},
		{"no aud, client_id is not allowed", nil, claims(jwt.MapClaims{"client_id": "someone-else"}), http.StatusUnauthorized},
		{"no aud and no client_id", nil, claims(nil), http.StatusUnauthorized},
		{"disallowed aud is not rescued by client_id", nil, claims(jwt.MapClaims{"aud": "https://other.example.com", "client_id": testOIDCClientID}), http.StatusUnauthorized},
		{"expired token with allowed aud", nil, claims(jwt.MapClaims{"aud": testOIDCClientID, "exp": time.Now().Add(-time.Hour).Unix()}), http.StatusUnauthorized},
		{"wrong issuer with allowed aud", nil, claims(jwt.MapClaims{"aud": testOIDCClientID, "iss": "https://evil.example.com"}), http.StatusUnauthorized},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine := newOIDCTestEngine(t, issuer, tc.audiences)

			w := get(t, engine, withBearer(sign(tc.claims)))

			require.Equal(t, tc.want, w.Code)
		})
	}

	t.Run("signature from a key the IdP does not publish", func(t *testing.T) {
		_, signElsewhere := oidcSigningServer(t)
		engine := newOIDCTestEngine(t, issuer, nil)

		w := get(t, engine, withBearer(signElsewhere(claims(jwt.MapClaims{"aud": testOIDCClientID}))))

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
