package packaging

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestMintToken(t *testing.T) {
	t.Parallel()

	const key = "test-key"

	tokenString, err := mintToken(key, defaultTokenTTL)
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

func TestResolveTokenTTL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requested string
		maxTTL    time.Duration
		want      time.Duration
		wantErr   error
	}{
		{"empty falls back to the default", "", 24 * time.Hour, defaultTokenTTL, nil},
		{"honors a requested lifetime", "15m", 24 * time.Hour, 15 * time.Minute, nil},
		{"honors a lifetime equal to the cap", "24h", 24 * time.Hour, 24 * time.Hour, nil},
		{"zero cap falls back to the default cap", "24h", 0, 24 * time.Hour, nil},
		{"rejects a lifetime above the cap", "24h", time.Hour, 0, ErrTokenTTLTooLong},
		{"rejects an unparseable lifetime", "fortnight", 24 * time.Hour, 0, ErrTokenTTLInvalid},
		{"rejects a non-positive lifetime", "0s", 24 * time.Hour, 0, ErrTokenTTLInvalid},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveTokenTTL(tt.requested, tt.maxTTL)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("ttl = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMintTokenUsesRequestedTTL(t *testing.T) {
	t.Parallel()

	const key = "test-key"

	tokenString, err := mintToken(key, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	claims := &jwt.RegisteredClaims{}

	if _, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	}); err != nil {
		t.Fatal(err)
	}

	got := time.Until(claims.ExpiresAt.Time)
	if got < 14*time.Minute || got > 15*time.Minute+time.Minute {
		t.Errorf("expiry in %v, want roughly 15m", got)
	}
}
