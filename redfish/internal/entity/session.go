// Package entity defines the Session data model for Redfish SessionService
package entity

import "time"

// Session represents a Redfish authentication session
// This entity bridges DMT Console's JWT authentication with Redfish SessionService.
type Session struct {
	// ID is the unique session identifier (UUID)
	ID string `json:"id"`

	// Username is the authenticated user
	Username string `json:"username"`

	// Token is the X-Auth-Token (JWT token from DMT Console)
	// This token can be validated using the existing JWT infrastructure
	Token string `json:"token"`

	// CreatedTime is when the session was created
	CreatedTime time.Time `json:"created_time"`

	// LastAccessTime tracks the last time this session was used
	LastAccessTime time.Time `json:"last_access_time"`

	// TimeoutSeconds is the session timeout in seconds
	TimeoutSeconds int `json:"timeout_seconds"`

	// ClientIP is the IP address of the client that created the session
	ClientIP string `json:"client_ip"`

	// UserAgent is the User-Agent header from the client
	UserAgent string `json:"user_agent"`

	// IsActive indicates if the session is still valid
	IsActive bool `json:"is_active"`
}

// IsExpired checks if the session has expired based on timeout.
func (s *Session) IsExpired() bool {
	if !s.IsActive {
		return true
	}

	timeout := time.Duration(s.TimeoutSeconds) * time.Second
	expirationTime := s.LastAccessTime.Add(timeout)

	return time.Now().After(expirationTime)
}

// Touch updates the last access time to keep session alive.
func (s *Session) Touch() {
	s.LastAccessTime = time.Now()
}

// Invalidate marks the session as inactive.
func (s *Session) Invalidate() {
	s.IsActive = false
}
