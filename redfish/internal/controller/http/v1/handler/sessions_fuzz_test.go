package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/redfish/internal/entity"
	redfishsessions "github.com/device-management-toolkit/console/redfish/internal/usecase/sessions"
)

type fuzzSessionRepo struct {
	sessions   map[string]*entity.Session
	tokenIndex map[string]string
}

func newFuzzSessionRepo() *fuzzSessionRepo {
	return &fuzzSessionRepo{
		sessions:   make(map[string]*entity.Session),
		tokenIndex: make(map[string]string),
	}
}

func (r *fuzzSessionRepo) Create(session *entity.Session) error {
	r.sessions[session.ID] = session
	r.tokenIndex[session.Token] = session.ID

	return nil
}

func (r *fuzzSessionRepo) Update(session *entity.Session) error {
	r.sessions[session.ID] = session
	r.tokenIndex[session.Token] = session.ID

	return nil
}

func (r *fuzzSessionRepo) Get(id string) (*entity.Session, error) {
	session, ok := r.sessions[id]
	if !ok {
		return nil, redfishsessions.ErrSessionNotFound
	}

	return session, nil
}

func (r *fuzzSessionRepo) GetByToken(token string) (*entity.Session, error) {
	id, ok := r.tokenIndex[token]
	if !ok {
		return nil, redfishsessions.ErrSessionNotFound
	}

	return r.Get(id)
}

func (r *fuzzSessionRepo) Delete(id string) error {
	session, ok := r.sessions[id]
	if !ok {
		return redfishsessions.ErrSessionNotFound
	}

	delete(r.tokenIndex, session.Token)
	delete(r.sessions, id)

	return nil
}

func (r *fuzzSessionRepo) List() ([]*entity.Session, error) {
	result := make([]*entity.Session, 0, len(r.sessions))

	for _, session := range r.sessions {
		result = append(result, session)
	}

	return result, nil
}

func (r *fuzzSessionRepo) DeleteExpired() (int, error) {
	removed := 0

	for id, session := range r.sessions {
		if session.IsExpired() {
			delete(r.tokenIndex, session.Token)
			delete(r.sessions, id)

			removed++
		}
	}

	return removed, nil
}

func setupFuzzSessionHandlerEnv() (*gin.Engine, *RedfishServer, *fuzzSessionRepo) {
	gin.SetMode(gin.TestMode)

	repo := newFuzzSessionRepo()
	cfg := &config.Config{
		Auth: config.Auth{
			AdminUsername: "admin",
			AdminPassword: "password",
			JWTKey:        "fuzz-test-secret-key-for-session-handlers",
			JWTExpiration: 24 * time.Hour,
		},
	}

	server := &RedfishServer{
		SessionUC: redfishsessions.NewUseCase(repo, cfg),
		Config:    cfg,
	}

	router := gin.New()
	router.POST("/redfish/v1/SessionService/Sessions", server.PostRedfishV1SessionServiceSessions)
	router.GET("/redfish/v1/SessionService", server.GetRedfishV1SessionService)
	router.GET("/redfish/v1/SessionService/Sessions", server.GetRedfishV1SessionServiceSessions)
	router.PATCH("/redfish/v1/SessionService", server.PatchRedfishV1SessionService)
	router.PUT("/redfish/v1/SessionService", server.PutRedfishV1SessionService)
	router.GET("/redfish/v1/SessionService/Sessions/:SessionId", func(c *gin.Context) {
		server.GetRedfishV1SessionServiceSessionsSessionId(c, c.Param("SessionId"))
	})
	router.DELETE("/redfish/v1/SessionService/Sessions/:SessionId", func(c *gin.Context) {
		server.DeleteRedfishV1SessionServiceSessionsSessionId(c, c.Param("SessionId"))
	})

	return router, server, repo
}

func FuzzPostSessionServiceSessions(f *testing.F) {
	seedInputs := []string{
		`{"UserName":"admin","Password":"password"}`,
		`{"UserName":"admin","Password":"wrong"}`,
		`{"UserName":"","Password":""}`,
		`{"UserName":"用戶🙂","Password":"päss\u0000секрет"}`,
		`{"UserName":123,"Password":true}`,
		`{"Password":"password"}`,
		`{"UserName":"admin"}`,
		`not-json`,
		`{}`,
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	f.Fuzz(func(t *testing.T, payload string) {
		router1, _, _ := setupFuzzSessionHandlerEnv()
		req1 := httptest.NewRequest(http.MethodPost, "/redfish/v1/SessionService/Sessions", bytes.NewBufferString(payload))
		req1.Header.Set("Content-Type", "application/json")

		w1 := httptest.NewRecorder()
		router1.ServeHTTP(w1, req1)

		router2, _, _ := setupFuzzSessionHandlerEnv()
		req2 := httptest.NewRequest(http.MethodPost, "/redfish/v1/SessionService/Sessions", bytes.NewBufferString(payload))
		req2.Header.Set("Content-Type", "application/json")

		w2 := httptest.NewRecorder()
		router2.ServeHTTP(w2, req2)

		if w1.Code != w2.Code {
			t.Fatalf("non-deterministic status for payload %q: first=%d second=%d", payload, w1.Code, w2.Code)
		}

		if w1.Code == http.StatusCreated {
			if w1.Header().Get(headerXAuthToken) == "" {
				t.Fatal("expected X-Auth-Token header on successful session creation")
			}

			if w1.Header().Get(headerLocation) == "" {
				t.Fatal("expected Location header on successful session creation")
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w1.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse successful response: %v", err)
			}
		}
	})
}

func FuzzSessionServiceMutationEndpoints(f *testing.F) {
	seedInputs := []string{
		`{}`,
		`{"ServiceEnabled":true}`,
		`{"SessionTimeout":1800}`,
		`{"SessionTimeout":"bad"}`,
		`{"junk":"value","nested":{"k":"v"}}`,
		`not-json`,
		`[]`,
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	f.Fuzz(func(t *testing.T, payload string) {
		router, _, _ := setupFuzzSessionHandlerEnv()

		for _, method := range []string{http.MethodPatch, http.MethodPut} {
			req := httptest.NewRequest(method, "/redfish/v1/SessionService", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			switch w.Code {
			case http.StatusOK, http.StatusBadRequest:
			default:
				t.Fatalf("unexpected status %d for %s payload %q", w.Code, method, payload)
			}
		}
	})
}

func FuzzSessionResourceHandlers(f *testing.F) {
	seedInputs := []string{
		"seed-session",
		"",
		"not-found",
		"用戶🙂",
		"../etc/passwd",
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	f.Fuzz(func(t *testing.T, sessionID string) {
		router, _, repo := setupFuzzSessionHandlerEnv()
		repo.sessions["seed-session"] = &entity.Session{
			ID:             "seed-session",
			Username:       "admin",
			Token:          "seed-token",
			CreatedTime:    time.Now(),
			LastAccessTime: time.Now(),
			TimeoutSeconds: 1800,
			IsActive:       true,
		}

		for _, method := range []string{http.MethodGet, http.MethodDelete} {
			req := httptest.NewRequest(method, "/redfish/v1/SessionService/Sessions/"+sessionID, http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			switch w.Code {
			case http.StatusOK,
				http.StatusNoContent,
				http.StatusBadRequest,
				http.StatusNotFound,
				http.StatusMovedPermanently,
				http.StatusTemporaryRedirect,
				http.StatusPermanentRedirect:
			default:
				t.Fatalf("unexpected status %d for %s sessionID %q", w.Code, method, sessionID)
			}
		}
	})
}

func buildAuthHeaderCase(modeName, xAuthToken, authHeader, validToken string, useRealToken bool) (tokenOut, authOut string) {
	if useRealToken {
		if modeName == "xauth" {
			return validToken, ""
		}

		return "", authBearerPrefix + validToken
	}

	return xAuthToken, authHeader
}

func runProtectedAuthRequest(protected *gin.Engine, xAuthToken, authHeader string) int {
	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)

	if xAuthToken != "" {
		req.Header.Set(headerXAuthToken, xAuthToken)
	}

	if authHeader != "" {
		req.Header.Set(headerAuthorization, authHeader)
	}

	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	return w.Code
}

func FuzzSessionAuthMiddleware(f *testing.F) {
	seedInputs := []struct {
		xAuthToken   string
		authHeader   string
		useRealToken bool
	}{
		{"", "", false},
		{"invalid-token", "", false},
		{"", "Bearer invalid-token", false},
		{"", "Basic abc", false},
		{"用戶🙂", "", false},
		{"", "Bearer ", true},
		{"", "", true},
	}

	for _, input := range seedInputs {
		f.Add(input.xAuthToken, input.authHeader, input.useRealToken)
	}

	f.Fuzz(func(t *testing.T, xAuthToken, authHeader string, useRealToken bool) {
		_, server, _ := setupFuzzSessionHandlerEnv()

		_, validToken, err := server.SessionUC.CreateSession("admin", "password", "127.0.0.1", "fuzzer")
		if err != nil {
			t.Fatalf("failed to create seed session: %v", err)
		}

		protected := gin.New()
		protected.GET("/protected", SessionAuthMiddleware(server.SessionUC), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		for _, mode := range []struct {
			name string
		}{
			{name: "xauth"},
			{name: "real"},
		} {
			headerToken, headerAuth := buildAuthHeaderCase(mode.name, xAuthToken, authHeader, validToken, useRealToken)
			statusCode := runProtectedAuthRequest(protected, headerToken, headerAuth)

			switch statusCode {
			case http.StatusOK, http.StatusUnauthorized:
			default:
				t.Fatalf("unexpected status %d for auth fuzz case", statusCode)
			}
		}
	})
}
