package packaging

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// defaultTokenTTL is how long a minted rpc-go auth token stays valid when the
	// request does not ask for a lifetime.
	defaultTokenTTL = time.Hour
	// defaultMaxTokenTTL caps a requested lifetime when the server sets no maximum.
	defaultMaxTokenTTL = 24 * time.Hour
)

var (
	// ErrTokenTTLInvalid indicates a token lifetime that is not a positive duration.
	ErrTokenTTLInvalid = errors.New("invalid token lifetime")
	// ErrTokenTTLTooLong indicates a token lifetime above the server maximum.
	ErrTokenTTLTooLong = errors.New("token lifetime exceeds the server maximum")
)

// resolveTokenTTL turns a requested lifetime into the duration to mint with.
// An empty request keeps defaultTokenTTL; a zero maxTTL falls back to
// defaultMaxTokenTTL so deployments predating the setting still have a ceiling.
func resolveTokenTTL(requested string, maxTTL time.Duration) (time.Duration, error) {
	if maxTTL <= 0 {
		maxTTL = defaultMaxTokenTTL
	}

	if requested == "" {
		return defaultTokenTTL, nil
	}

	ttl, err := time.ParseDuration(requested)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrTokenTTLInvalid, requested)
	}

	if ttl <= 0 {
		return 0, fmt.Errorf("%w: %q", ErrTokenTTLInvalid, requested)
	}

	if ttl > maxTTL {
		return 0, fmt.Errorf("%w of %s: %q", ErrTokenTTLTooLong, maxTTL, requested)
	}

	return ttl, nil
}

// mintToken issues an HS256 JWT signed with the given key, mirroring the
// login route's token issuance. rpc-go uses this as its bearer auth-token.
func mintToken(jwtKey string, ttl time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(jwtKey))
}
