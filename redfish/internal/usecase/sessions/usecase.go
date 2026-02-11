// Package sessions provides business logic for Redfish SessionService
package sessions

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/redfish/internal/entity"
)

const (
	// DefaultSessionTimeout is the default session timeout in seconds (30 minutes).
	DefaultSessionTimeout = 1800
)

// UseCase defines the session management business logic.
type UseCase struct {
	repo           Repository
	config         *config.Config
	sessionTimeout int // seconds
}

// NewUseCase creates a new session use case.
func NewUseCase(repo Repository, cfg *config.Config) *UseCase {
	return &UseCase{
		repo:           repo,
		config:         cfg,
		sessionTimeout: DefaultSessionTimeout,
	}
}

// CreateSession creates a new session with JWT token.
// This integrates with DMT Console's existing JWT authentication.
// If a session already exists for this user, it returns an error to prevent multiple concurrent sessions.
func (uc *UseCase) CreateSession(username, password, clientIP, userAgent string) (*entity.Session, string, error) {
	// Validate credentials using DMT Console's admin credentials
	if username != uc.config.AdminUsername || password != uc.config.AdminPassword {
		return nil, "", ErrInvalidCredentials
	}

	// Check if an active session already exists for this user
	existingSessions, err := uc.repo.List()
	if err != nil {
		return nil, "", fmt.Errorf("failed to check existing sessions: %w", err)
	}

	for _, session := range existingSessions {
		if session.Username == username && session.IsActive {
			// Return error - cannot create duplicate session
			return nil, "", ErrSessionAlreadyExists
		}
	}

	// Generate unique session ID
	sessionID := uuid.New().String()

	// Create JWT token (reusing DMT Console's JWT infrastructure)
	expirationTime := time.Now().Add(uc.config.JWTExpiration)
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		Subject:   username,
		ID:        sessionID, // Include session ID in JWT
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	jwtToken, err := token.SignedString([]byte(uc.config.JWTKey))
	if err != nil {
		return nil, "", err
	}

	// Create session entity
	session := &entity.Session{
		ID:             sessionID,
		Username:       username,
		Token:          jwtToken,
		CreatedTime:    time.Now(),
		LastAccessTime: time.Now(),
		TimeoutSeconds: uc.sessionTimeout,
		ClientIP:       clientIP,
		UserAgent:      userAgent,
		IsActive:       true,
	}

	// Store session in repository
	if err := uc.repo.Create(session); err != nil {
		return nil, "", err
	}

	return session, jwtToken, nil
}

// ValidateToken validates a session token (JWT).
// This can work in two modes:
// 1. Stateless: Just validate JWT signature and expiration.
// 2. Stateful: Also check if session exists and is active.
func (uc *UseCase) ValidateToken(tokenString string) (*entity.Session, error) {
	// Parse and validate JWT
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
		return []byte(uc.config.JWTKey), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check if session exists and is active (stateful check)
	session, err := uc.repo.GetByToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Update last access time
	session.Touch()

	if err := uc.repo.Update(session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by ID.
func (uc *UseCase) GetSession(sessionID string) (*entity.Session, error) {
	return uc.repo.Get(sessionID)
}

// DeleteSession terminates a session (logout).
func (uc *UseCase) DeleteSession(sessionID string) error {
	return uc.repo.Delete(sessionID)
}

// ListSessions returns all active sessions.
func (uc *UseCase) ListSessions() ([]*entity.Session, error) {
	return uc.repo.List()
}

// GetSessionCount returns the number of active sessions.
func (uc *UseCase) GetSessionCount() (int, error) {
	sessions, err := uc.repo.List()
	if err != nil {
		return 0, err
	}

	return len(sessions), nil
}

var ErrInvalidCredentials = ErrInvalidToken // ErrInvalidCredentials is returned when credentials are invalid.
