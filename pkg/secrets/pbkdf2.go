package secrets

import (
	"bytes"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

const (
	pbkdf2Prefix     = "$pbkdf2$"
	pbkdf2Iterations = 100000
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
	if !IsPBKDF2Hash(hashedPassword) {
		return false
	}

	// Parse format: $pbkdf2$<iterations>$<hex(salt)>$<hex(hash)>
	parts := strings.Split(strings.TrimPrefix(hashedPassword, pbkdf2Prefix), "$")
	if len(parts) != pbkdf2PartsCount {
		return false
	}

	iterations, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}

	salt, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}

	stored, err := hex.DecodeString(parts[2])
	if err != nil {
		return false
	}

	computed, err := pbkdf2.Key(sha256.New, password, salt, iterations, pbkdf2HashSize)
	if err != nil {
		return false
	}

	return bytes.Equal(computed, stored)
}

// IsPBKDF2Hash checks if a value is already a PBKDF2 hash.
func IsPBKDF2Hash(value string) bool {
	return strings.HasPrefix(value, pbkdf2Prefix)
}
