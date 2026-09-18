package port

import (
	"context"

	"github.com/WilsonSayago/authBase/v3/core/domain"
)

// RefreshTokenStore persists refresh sessions by token hash and rotates them
// atomically. Implementations must never store the raw JWT.
type RefreshTokenStore interface {
	Create(ctx context.Context, session domain.RefreshSession) error
	// Rotate consumes the session identified by currentHash and creates next in
	// one transaction. Concurrent callers: exactly one succeeds.
	Rotate(ctx context.Context, currentHash [32]byte, next domain.RefreshSession) error
	RevokeFamily(ctx context.Context, familyID string) error
}
