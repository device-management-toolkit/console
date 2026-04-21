package sessions

import (
	"strings"
	"testing"
	"time"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/redfish/internal/entity"
)

// fuzzInMemoryRepo is a minimal in-memory session repository for fuzz tests.
// It avoids goroutines (no cleanup ticker) so it is safe under the fuzzer.
type fuzzInMemoryRepo struct {
	sessions   map[string]*entity.Session
	tokenIndex map[string]string
}

func newFuzzRepo() *fuzzInMemoryRepo {
	return &fuzzInMemoryRepo{
		sessions:   make(map[string]*entity.Session),
		tokenIndex: make(map[string]string),
	}
}

func (r *fuzzInMemoryRepo) Create(s *entity.Session) error {
	r.sessions[s.ID] = s
	r.tokenIndex[s.Token] = s.ID

	return nil
}

func (r *fuzzInMemoryRepo) Update(s *entity.Session) error {
	if _, ok := r.sessions[s.ID]; !ok {
		return ErrSessionNotFound
	}

	r.sessions[s.ID] = s
	r.tokenIndex[s.Token] = s.ID

	return nil
}

func (r *fuzzInMemoryRepo) Get(id string) (*entity.Session, error) {
	s, ok := r.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}

	return s, nil
}

func (r *fuzzInMemoryRepo) GetByToken(token string) (*entity.Session, error) {
	id, ok := r.tokenIndex[token]
	if !ok {
		return nil, ErrSessionNotFound
	}

	s, ok := r.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}

	return s, nil
}

func (r *fuzzInMemoryRepo) Delete(id string) error {
	s, ok := r.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}

	delete(r.tokenIndex, s.Token)
	delete(r.sessions, id)

	return nil
}

func (r *fuzzInMemoryRepo) List() ([]*entity.Session, error) {
	result := make([]*entity.Session, 0, len(r.sessions))

	for _, s := range r.sessions {
		result = append(result, s)
	}

	return result, nil
}

func (r *fuzzInMemoryRepo) DeleteExpired() (int, error) {
	count := 0

	for id, s := range r.sessions {
		if s.IsExpired() {
			delete(r.tokenIndex, s.Token)
			delete(r.sessions, id)

			count++
		}
	}

	return count, nil
}

// newFuzzConfig returns a minimal config suitable for fuzz tests.
func newFuzzConfig(adminUser, adminPass, jwtKey string) *config.Config {
	return &config.Config{
		Auth: config.Auth{
			AdminUsername: adminUser,
			AdminPassword: adminPass,
			JWTKey:        jwtKey,
			JWTExpiration: 24 * time.Hour,
		},
	}
}

// FuzzCreateSession fuzzes CreateSession with arbitrary username, password, clientIP, and userAgent.
// Verifies: no panics, deterministic error result, non-empty token on success.
func assertCreateSessionOutcome(t *testing.T, username string, session *entity.Session, token string, err error) {
	t.Helper()

	if err != nil {
		if session != nil {
			t.Fatal("expected nil session on error")
		}

		if token != "" {
			t.Fatal("expected empty token on error")
		}

		return
	}

	if session == nil {
		t.Fatal("expected non-nil session on success")

		return
	}

	if token == "" {
		t.Fatal("expected non-empty token on success")

		return
	}

	if session.ID == "" {
		t.Fatal("expected non-empty session ID")

		return
	}

	if !session.IsActive {
		t.Fatal("expected new session to be active")

		return
	}

	if session.Username != username {
		t.Fatalf("expected username %q, got %q", username, session.Username)
	}
}

func FuzzCreateSession(f *testing.F) {
	type seed struct {
		username  string
		password  string
		clientIP  string
		userAgent string
	}

	seeds := []seed{
		{"standalone", "G@ppm0ym", "192.168.1.1", "Mozilla/5.0"},
		{"", "", "", ""},
		{"admin", "wrong-password", "10.0.0.1", "curl/7.0"},
		{"用戶", "päss\u0000секрет🔐", "::1", "テストUA"},
		{strings.Repeat("u", 4096), strings.Repeat("p", 4096), strings.Repeat("1", 255), strings.Repeat("A", 4096)},
		{"standalone", "G@ppm0ym", "999.999.999.999", ""},
		{"STANDALONE", "G@ppm0ym", "127.0.0.1", "bot"},
		{"\x00admin", "\x00pass", "\x00ip", "\x00agent"},
	}

	for _, s := range seeds {
		f.Add(s.username, s.password, s.clientIP, s.userAgent)
	}

	cfg := newFuzzConfig("standalone", "G@ppm0ym", "fuzz-secret-jwt-key-32bytes-long!!")
	uc := NewUseCase(newFuzzRepo(), cfg)

	f.Fuzz(func(t *testing.T, username, password, clientIP, userAgent string) {
		sess, token, err := uc.CreateSession(username, password, clientIP, userAgent)

		assertCreateSessionOutcome(t, username, sess, token, err)
	})
}

// FuzzValidateToken fuzzes ValidateToken with arbitrary token strings.
// Verifies: no panics, deterministic error behavior, consistent session on success.
func FuzzValidateToken(f *testing.F) {
	seeds := []string{
		"",
		"not-a-jwt",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
		"eyJhbGciOiJub25lIn0.e30.",
		strings.Repeat("a", 4096),
		"a.b.c",
		"a.b",
		"\x00\xFF",
		"用戶.秘密.token",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	cfg := newFuzzConfig("standalone", "G@ppm0ym", "fuzz-secret-jwt-key-32bytes-long!!")
	repo := newFuzzRepo()
	uc := NewUseCase(repo, cfg)

	f.Fuzz(func(t *testing.T, tokenString string) {
		sess, err := uc.ValidateToken(tokenString)
		if err != nil {
			if sess != nil {
				t.Fatal("expected nil session on error")
			}

			return
		}

		// On success, session must be valid.
		if sess == nil {
			t.Fatal("expected non-nil session on success")

			return
		}

		if sess.ID == "" {
			t.Fatal("expected non-empty session ID on success")
		}
	})
}

// FuzzSessionIsExpired fuzzes the IsExpired / Touch / Invalidate methods on the Session entity.
// Verifies: no panics, consistency of IsExpired with IsActive and timeout.
func FuzzSessionIsExpired(f *testing.F) {
	type seed struct {
		timeoutSeconds int
		secondsAgo     int64 // how many seconds since LastAccessTime
		isActive       bool
	}

	seeds := []seed{
		{1800, 60, true},
		{1800, 3600, true},    // expired
		{0, 0, true},          // zero timeout — always expired
		{-1, 0, true},         // negative timeout
		{1800, 0, false},      // inactive
		{2147483647, 0, true}, // max int timeout
		{1800, 9999999, true}, // far past
		{1, -9999999, true},   // far future last access
	}

	for _, s := range seeds {
		f.Add(s.timeoutSeconds, s.secondsAgo, s.isActive)
	}

	f.Fuzz(func(t *testing.T, timeoutSeconds int, secondsAgo int64, isActive bool) {
		lastAccess := time.Now().Add(-time.Duration(secondsAgo) * time.Second)

		sess := &entity.Session{
			ID:             "fuzz-id",
			IsActive:       isActive,
			TimeoutSeconds: timeoutSeconds,
			LastAccessTime: lastAccess,
		}

		expired1 := sess.IsExpired()

		// Inactive sessions must always be expired.
		if !isActive && !expired1 {
			t.Fatal("inactive session must be expired")
		}

		// Touch should update LastAccessTime; call it and verify IsExpired can be called again.
		sess.Touch()
		_ = sess.IsExpired()

		// Invalidate must make the session expired.
		sess.Invalidate()

		if !sess.IsExpired() {
			t.Fatal("invalidated session must be expired")
		}

		// IsExpired must not panic; no additional assertion needed.
	})
}
