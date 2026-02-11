// Package sessions provides session management use cases for Redfish SessionService
package sessions

import (
	"errors"

	"github.com/device-management-toolkit/console/redfish/internal/entity"
)

var (
	// ErrSessionNotFound is returned when a session cannot be found.
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionExpired is returned when a session has expired.
	ErrSessionExpired = errors.New("session expired")

	// ErrInvalidToken is returned when a token is invalid.
	ErrInvalidToken = errors.New("invalid token")

	// ErrSessionAlreadyExists is returned when trying to create a session for a user who already has an active session.
	ErrSessionAlreadyExists = errors.New("an active session already exists for this user")
)

// Repository defines the interface for session storage.
type Repository interface {
	// Create stores a new session
	Create(session *entity.Session) error

	// Update modifies an existing session
	Update(session *entity.Session) error

	// Get retrieves a session by ID
	Get(id string) (*entity.Session, error)

	// GetByToken retrieves a session by token
	GetByToken(token string) (*entity.Session, error)

	// Delete removes a session
	Delete(id string) error

	// List returns all active sessions
	List() ([]*entity.Session, error)

	// DeleteExpired removes all expired sessions
	DeleteExpired() (int, error)
}
