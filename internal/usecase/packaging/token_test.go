package packaging

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestMintToken(t *testing.T) {
	t.Parallel()

	const key = "test-key"

	tokenString, err := mintToken(key)
	if err != nil {
		t.Fatal(err)
	}

	if tokenString == "" {
		t.Fatal("expected a non-empty token")
	}

	claims := &jwt.RegisteredClaims{}

	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(tok *jwt.Token) (interface{}, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}

		return []byte(key), nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if !parsed.Valid {
		t.Fatal("expected a valid token")
	}

	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		t.Fatal("expected an expiry in the future")
	}
}
