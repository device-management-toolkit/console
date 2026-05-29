package packaging

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// tokenTTL is how long a minted rpc-go auth token stays valid.
const tokenTTL = time.Hour

// mintToken issues an HS256 JWT signed with the given key, mirroring the
// login route's token issuance. rpc-go uses this as its bearer auth-token.
func mintToken(jwtKey string) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(jwtKey))
}
