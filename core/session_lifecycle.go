package core

import "context"

// SessionIssuer establishes a persisted token session for an identity that the
// caller has already authenticated. Implementations must still verify that the
// identity exists and is active before issuing tokens.
type SessionIssuer interface {
	EstablishSession(ctx context.Context, userID string) (accessToken, refreshToken string, err error)
}

// SessionManager revokes all refresh sessions owned by one user.
type SessionManager interface {
	RevokeUserSessions(ctx context.Context, userID string) error
}
