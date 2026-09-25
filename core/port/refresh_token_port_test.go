package port_test

import (
	"context"

	"github.com/WilsonSayago/authBase/v4/core/domain"
	"github.com/WilsonSayago/authBase/v4/core/port"
)

type legacyV4RefreshStore struct{}

func (legacyV4RefreshStore) Create(context.Context, domain.RefreshSession) error {
	return nil
}

func (legacyV4RefreshStore) Rotate(context.Context, [32]byte, domain.RefreshSession) error {
	return nil
}

func (legacyV4RefreshStore) RevokeFamily(context.Context, string) error {
	return nil
}

// Compile-time coverage: additive session capabilities must not become
// mandatory methods on the original v4 RefreshTokenStore.
var _ port.RefreshTokenStore = legacyV4RefreshStore{}
