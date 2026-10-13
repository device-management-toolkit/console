package secrets

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

const (
	pbkdf2Prefix     = "$pbkdf2$"
	pbkdf2Iterations = 600000
	pbkdf2SaltSize   = 16
	pbkdf2HashSize   = 32
	pbkdf2PartsCount = 3
)

// GeneratePBKDF2Hash generates a PBKDF2 hash from a password string.
// The resulting hash includes the iterations, salt, and hash in the format:
// $pbkdf2$<iterations>$<hex(salt)>$<hex(hash)>.
func GeneratePBKDF2Hash(password string) (string, error) {
	salt := make([]byte, pbkdf2SaltSize)

	_, err := rand.Read(salt)
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iterations, pbkdf2HashSize)
	if err != nil {
		return "", fmt.Errorf("failed to generate PBKDF2 hash: %w", err)
	}

	// Format: $pbkdf2$<iterations>$<hex(salt)>$<hex(hash)>
	formatted := fmt.Sprintf("%s%d$%s$%s",
		pbkdf2Prefix,
		pbkdf2Iterations,
		hex.EncodeToString(salt),
		hex.EncodeToString(hash),
	)

	return formatted, nil
}

// VerifyPBKDF2Hash verifies a password against a PBKDF2 hash.
// Returns true if the password matches the hash, false otherwise.
func VerifyPBKDF2Hash(hashedPassword, password string) bool {
	parsed, ok := parsePBKDF2Hash(hashedPassword)
	if !ok {
		return false
	}

	computed, err := pbkdf2.Key(sha256.New, password, parsed.salt, parsed.iterations, pbkdf2HashSize)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(computed, parsed.hash) == 1
}

// IsPBKDF2Hash reports whether value is a fully well-formed PBKDF2 hash
// produced by GeneratePBKDF2Hash, not merely prefixed with pbkdf2Prefix. A
// plaintext password that happens to start with the prefix must be rejected
// here, otherwise it would be stored verbatim as if already hashed and could
// never verify again.
func IsPBKDF2Hash(value string) bool {
	_, ok := parsePBKDF2Hash(value)

	return ok
}

// parsedPBKDF2Hash holds the decoded fields of a $pbkdf2$<iterations>$<hex(salt)>$<hex(hash)> value.
type parsedPBKDF2Hash struct {
	iterations int
	salt       []byte
	hash       []byte
}

// parsePBKDF2Hash parses and validates the $pbkdf2$<iterations>$<hex(salt)>$<hex(hash)>
// format, returning ok=false for anything that doesn't fully match.
func parsePBKDF2Hash(value string) (parsedPBKDF2Hash, bool) {
	if !strings.HasPrefix(value, pbkdf2Prefix) {
		return parsedPBKDF2Hash{}, false
	}

	parts := strings.Split(strings.TrimPrefix(value, pbkdf2Prefix), "$")
	if len(parts) != pbkdf2PartsCount {
		return parsedPBKDF2Hash{}, false
	}

	iterations, err := strconv.Atoi(parts[0])
	if err != nil || iterations <= 0 {
		return parsedPBKDF2Hash{}, false
	}

	salt, err := hex.DecodeString(parts[1])
	if err != nil || len(salt) != pbkdf2SaltSize {
		return parsedPBKDF2Hash{}, false
	}

	hash, err := hex.DecodeString(parts[2])
	if err != nil || len(hash) != pbkdf2HashSize {
		return parsedPBKDF2Hash{}, false
	}

	return parsedPBKDF2Hash{iterations: iterations, salt: salt, hash: hash}, true
}
