package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/secrets"
)

// resetLoginAttempts clears the package-global rate-limit state so each test
// starts clean, and restores the original maxFailedLoginAttempts/
// rateLimitWindow afterward for tests that tune them.
func resetLoginAttempts(t *testing.T) {
	t.Helper()

	origMax := maxFailedLoginAttempts
	origWindow := rateLimitWindow

	loginMutex.Lock()
	loginAttempts = make(map[string]*loginAttemptEntry)
	loginMutex.Unlock()

	t.Cleanup(func() {
		loginMutex.Lock()
		loginAttempts = make(map[string]*loginAttemptEntry)
		loginMutex.Unlock()

		maxFailedLoginAttempts = origMax
		rateLimitWindow = origWindow
	})
}

// rateLimitTestEngine wires a basic-auth login route with a known bad
// password, relying on gin's default trust-all-proxies behavior (SetTrustedProxies
// is only locked down by setupHTTPHandler in internal/app, not here) so each
// test can simulate a distinct client via X-Forwarded-For.
func rateLimitTestEngine(t *testing.T) *gin.Engine {
	t.Helper()

	hash, err := secrets.GeneratePBKDF2Hash(testAdminPass)
	require.NoError(t, err)

	cfg := &config.Config{}
	cfg.AdminUsername = testAdminUser
	cfg.AdminPassword = hash
	cfg.JWTKey = testJWTKey
	cfg.JWTExpiration = time.Hour

	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	config.ConsoleConfig = cfg

	engine := gin.New()
	route := LoginRoute{Config: cfg}
	engine.POST(testAuthorizeURL, route.Login)

	return engine
}

func attemptLogin(t *testing.T, engine *gin.Engine, clientIP, password string) *httptest.ResponseRecorder {
	t.Helper()

	body := `{"username":"` + testAdminUser + `","password":"` + password + `"}`

	req, err := http.NewRequest(http.MethodPost, testAuthorizeURL, bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", clientIP)
	// ClientIP() needs a parseable RemoteAddr to decide whether to trust the
	// X-Forwarded-For header at all; http.NewRequest leaves it empty.
	req.RemoteAddr = "198.51.100.1:12345"

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	return w
}

// seedExhaustedEntry puts clientIP one reservation away from (or already at)
// the limit, avoiding maxFailedLoginAttempts real PBKDF2-hashing requests
// just to reach that state in tests that care about what happens at/after it.
func seedEntry(clientIP string, count int) {
	loginMutex.Lock()
	loginAttempts[clientIP] = &loginAttemptEntry{count: count, windowStart: time.Now()}
	loginMutex.Unlock()
}

//nolint:paralleltest // mutates the package-global loginAttempts map
func TestRateLimit_BlocksAfterMaxFailedAttempts(t *testing.T) {
	resetLoginAttempts(t)

	engine := rateLimitTestEngine(t)
	clientIP := "203.0.113.1"

	seedEntry(clientIP, maxFailedLoginAttempts-1)

	// The last reservation within the limit still runs the real handler path.
	w := attemptLogin(t, engine, clientIP, "wrong-password")
	require.Equal(t, http.StatusUnauthorized, w.Code, "the attempt that reaches the limit is still evaluated")

	w = attemptLogin(t, engine, clientIP, "wrong-password")
	require.Equal(t, http.StatusTooManyRequests, w.Code, "attempt beyond the limit must be rate limited")

	var got map[string]string

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "rate_limited", got["error"])

	// Even a correct password is rejected once the limit is reached.
	w = attemptLogin(t, engine, clientIP, testAdminPass)
	require.Equal(t, http.StatusTooManyRequests, w.Code, "rate limit applies regardless of credential correctness")
}

//nolint:paralleltest // mutates the package-global loginAttempts map
func TestRateLimit_PerClientIsolation(t *testing.T) {
	resetLoginAttempts(t)

	engine := rateLimitTestEngine(t)

	seedEntry("203.0.113.5", maxFailedLoginAttempts)
	require.Equal(t, http.StatusTooManyRequests, attemptLogin(t, engine, "203.0.113.5", "wrong-password").Code)

	// A different client IP must not be affected by the first client's lockout.
	w := attemptLogin(t, engine, "203.0.113.6", testAdminPass)
	require.Equal(t, http.StatusOK, w.Code)
}

//nolint:paralleltest // mutates the package-global loginAttempts map
func TestRateLimit_SuccessfulLoginResetsCounter(t *testing.T) {
	resetLoginAttempts(t)

	engine := rateLimitTestEngine(t)
	clientIP := "203.0.113.2"

	seedEntry(clientIP, maxFailedLoginAttempts-1)

	// One attempt below the limit remains; a successful login should reset it.
	require.Equal(t, http.StatusOK, attemptLogin(t, engine, clientIP, testAdminPass).Code)

	loginMutex.Lock()
	_, stillTracked := loginAttempts[clientIP]
	loginMutex.Unlock()
	require.False(t, stillTracked, "a successful login must clear the client's attempt counter")

	// With the counter cleared, the client gets a fresh allowance rather than
	// being treated as already at (one below) the limit.
	w := attemptLogin(t, engine, clientIP, "wrong-password")
	require.Equal(t, http.StatusUnauthorized, w.Code, "counter should have restarted after the successful login")
}

//nolint:paralleltest // mutates the package-global rateLimitWindow/loginAttempts
func TestRateLimit_ExpiredWindowResetsCounter(t *testing.T) {
	resetLoginAttempts(t)

	rateLimitWindow = 10 * time.Millisecond

	engine := rateLimitTestEngine(t)
	clientIP := "203.0.113.3"

	// Seed an already-exhausted entry directly rather than driving
	// maxFailedLoginAttempts real (slow, PBKDF2-hashing) requests through it.
	loginMutex.Lock()
	loginAttempts[clientIP] = &loginAttemptEntry{count: maxFailedLoginAttempts, windowStart: time.Now()}
	loginMutex.Unlock()

	require.Equal(t, http.StatusTooManyRequests, attemptLogin(t, engine, clientIP, testAdminPass).Code,
		"still within the window, so the exhausted entry should still block")

	time.Sleep(20 * time.Millisecond)

	// The window has elapsed, so the client gets a fresh allowance.
	w := attemptLogin(t, engine, clientIP, testAdminPass)
	require.Equal(t, http.StatusOK, w.Code, "expired window should have reset the attempt count")
}

//nolint:paralleltest // mutates the package-global loginAttempts map
func TestRateLimit_ConcurrentReservationsAreSerialized(t *testing.T) {
	resetLoginAttempts(t)

	engine := rateLimitTestEngine(t)
	clientIP := "203.0.113.4"

	const concurrentRequests = 20

	var (
		wg          sync.WaitGroup
		mu          sync.Mutex
		allowed     int
		rateLimited int
	)

	wg.Add(concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func() {
			defer wg.Done()

			w := attemptLogin(t, engine, clientIP, "wrong-password")

			mu.Lock()
			defer mu.Unlock()

			switch w.Code {
			case http.StatusUnauthorized:
				allowed++
			case http.StatusTooManyRequests:
				rateLimited++
			}
		}()
	}

	wg.Wait()

	require.Equal(t, maxFailedLoginAttempts, allowed, "exactly the configured number of attempts must be reserved despite concurrency")
	require.Equal(t, concurrentRequests-maxFailedLoginAttempts, rateLimited)
}
