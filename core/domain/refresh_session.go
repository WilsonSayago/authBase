package domain

import (
	"crypto/sha256"
	"time"
)

// RefreshSession is the persisted refresh-token metadata. Stores must keep
// TokenHash only — never the raw JWT.
type RefreshSession struct {
	TokenID   string
	FamilyID  string
	UserID    string
	TokenHash [32]byte
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// HashRefreshToken returns SHA-256 of the full serialized refresh JWT.
func HashRefreshToken(token string) [32]byte {
	return sha256.Sum256([]byte(token))
}

// Valid reports whether the session has the required identifiers and lifetime.
func (s RefreshSession) Valid() bool {
	if s.TokenID == "" || s.FamilyID == "" || s.UserID == "" {
		return false
	}
	if s.IssuedAt.IsZero() || s.ExpiresAt.IsZero() {
		return false
	}
	return s.ExpiresAt.After(s.IssuedAt)
}
