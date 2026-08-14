package config

import (
	"crypto/aes"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
)

// Errors returned by ValidateEncryptionKey.
var (
	// ErrEncryptionKeyLength means crypto/aes cannot use the key at all — every
	// credential write would fail at runtime with "invalid key size".
	ErrEncryptionKeyLength = errors.New("encryption key must be 16, 24 or 32 characters long (AES-128, AES-192 or AES-256)")
	// ErrEncryptionKeyWeak means the key is the right size but so predictable
	// that it contributes little to the strength of the encryption.
	ErrEncryptionKeyWeak = errors.New("encryption key is too weak")
)

const (
	// Key sizes crypto/aes accepts, in bytes.
	aes128KeySize = 16
	aes192KeySize = 24
	aes256KeySize = 32

	// minDistinctChars is the smallest number of distinct characters a key may
	// be built from. The generated key (24 random bytes, base64-encoded to 32
	// characters) averages ~25.
	minDistinctChars = 8

	// minEntropyBitsPerChar is the Shannon entropy floor across the key's own
	// characters. "aaaaaaaaaaaaaaaa" scores 0, the generated key ~4.6, and a
	// random 16-character key ~3.75, so the floor rejects repetitive keys
	// without rejecting anything genuinely random.
	minEntropyBitsPerChar = 3.0

	// maxSequentialRun caps runs of consecutive characters ("012345", "abcdef").
	// Such a run is vanishingly unlikely in a random key and typical of a key
	// typed by hand.
	maxSequentialRun = 6
)

// ValidateEncryptionKey reports whether key can be used as the AES-GCM key that
// protects device credentials. It enforces the size crypto/aes requires plus a
// minimum strength, so a key with no entropy ("aaaaaaaaaaaaaaaa") cannot silently
// negate the encryption.
//
// Callers are responsible for skipping the empty key, which means "no key
// supplied" rather than "invalid key" — see handleEncryptionKey in cmd/app.
func ValidateEncryptionKey(key string) error {
	if _, err := aes.NewCipher(encryptionKeyBytes(key)); err != nil {
		return fmt.Errorf("%w: %v", ErrEncryptionKeyLength, err)
	}

	if distinct := distinctChars(key); distinct < minDistinctChars {
		return fmt.Errorf("%w: built from only %d distinct characters, need at least %d", ErrEncryptionKeyWeak, distinct, minDistinctChars)
	}

	if bits := entropyBitsPerChar(key); bits < minEntropyBitsPerChar {
		return fmt.Errorf("%w: too repetitive (%.1f bits of entropy per character, need %.1f)", ErrEncryptionKeyWeak, bits, minEntropyBitsPerChar)
	}

	if period := repeatPeriod(key); period > 0 {
		return fmt.Errorf("%w: repeats the same %d characters", ErrEncryptionKeyWeak, period)
	}

	if run := longestSequentialRun(key); run >= maxSequentialRun {
		return fmt.Errorf("%w: contains a run of %d sequential characters", ErrEncryptionKeyWeak, run)
	}

	return nil
}

func encryptionKeyBytes(key string) []byte {
	if len(key) == 44 {
		if decoded, err := base64.StdEncoding.DecodeString(key); err == nil && len(decoded) == aes256KeySize {
			return decoded
		}
	}

	return []byte(key)
}

// distinctChars counts how many different bytes the key is built from.
func distinctChars(key string) int {
	seen := make(map[byte]struct{}, len(key))
	for _, c := range []byte(key) {
		seen[c] = struct{}{}
	}

	return len(seen)
}

// entropyBitsPerChar is the Shannon entropy of the key's own character
// distribution. It measures repetition within the key, not the size of the
// alphabet it was drawn from.
func entropyBitsPerChar(key string) float64 {
	counts := make(map[byte]int, len(key))
	for _, c := range []byte(key) {
		counts[c]++
	}

	length := float64(len(key))

	var bits float64

	for _, count := range counts {
		p := float64(count) / length
		bits -= p * math.Log2(p)
	}

	return bits
}

// repeatPeriod returns the length of the shortest block the key is a repetition
// of ("aaaa" → 1, "abcabc" → 3), or 0 when the key does not repeat within its
// first half.
func repeatPeriod(key string) int {
	for period := 1; period+period <= len(key); period++ {
		if isPeriodic(key, period) {
			return period
		}
	}

	return 0
}

// isPeriodic reports whether key repeats every period characters.
func isPeriodic(key string, period int) bool {
	for i := period; i < len(key); i++ {
		if key[i] != key[i-period] {
			return false
		}
	}

	return true
}

// longestSequentialRun returns the length of the longest run of characters that
// each step one code point up or down from the previous one.
func longestSequentialRun(key string) int {
	if key == "" {
		return 0
	}

	longest, current, step := 1, 1, 0

	for i := 1; i < len(key); i++ {
		diff := int(key[i]) - int(key[i-1])

		switch {
		case diff != 1 && diff != -1:
			current, step = 1, 0
		case current == 1 || diff == step:
			current++
			step = diff
		default:
			current, step = 2, diff
		}

		longest = max(longest, current)
	}

	return longest
}
