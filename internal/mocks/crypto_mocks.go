package mocks

import (
	crypto "github.com/device-management-toolkit/console/internal/mocks/crypto"
)

// MockCrypto is a fake security.Cryptor for tests. The implementation lives in
// the internal/mocks/crypto leaf package so it can be shared with internal
// (white-box) test files without an import cycle; this alias preserves the
// existing mocks.MockCrypto call sites.
type MockCrypto = crypto.MockCrypto
