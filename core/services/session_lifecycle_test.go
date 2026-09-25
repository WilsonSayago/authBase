package services

import (
	"context"
	"errors"
	"testing"

	"github.com/WilsonSayago/authBase/v4/core"
	"github.com/WilsonSayago/authBase/v4/core/domain"
	"github.com/WilsonSayago/authBase/v4/core/port"
)

type baselineRefreshStore struct {
	store *fakeRefreshStore
}

func (s baselineRefreshStore) Create(ctx context.Context, session domain.RefreshSession) error {
	return s.store.Create(ctx, session)
}

func (s baselineRefreshStore) Rotate(ctx context.Context, currentHash [32]byte, next domain.RefreshSession) error {
	return s.store.Rotate(ctx, currentHash, next)
}

func (s baselineRefreshStore) RevokeFamily(ctx context.Context, familyID string) error {
	return s.store.RevokeFamily(ctx, familyID)
}

var (
	_ port.RefreshTokenStore = baselineRefreshStore{}
	_ core.SessionIssuer     = (*AuthenticationService[fakeUser])(nil)
	_ core.SessionManager    = (*AuthenticationService[fakeUser])(nil)
)

func TestEstablishSessionPersistsForActiveUser(t *testing.T) {
	t.Parallel()

	identities := newFakeIdentityStore()
	identities.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	refresh := newFakeRefreshStore()
	svc := newTestAuthServiceWithRefresh(t, identities, refresh, &fakeValidationPort{})

	access, refreshToken, err := svc.EstablishSession(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("EstablishSession() error = %v", err)
	}
	if access == "" || refreshToken == "" {
		t.Fatal("EstablishSession() returned empty tokens")
	}
	if identities.findByIDN != 1 {
		t.Fatalf("FindByID calls = %d, want 1", identities.findByIDN)
	}
	if refresh.createN != 1 {
		t.Fatalf("Create calls = %d, want 1", refresh.createN)
	}
}

func TestEstablishSessionRejectsUnavailableIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		userID string
		setup  func(*fakeIdentityStore)
		want   error
	}{
		{
			name:   "inactive",
			userID: "user-1",
			setup: func(store *fakeIdentityStore) {
				store.add(fakeUser{id: "user-1", email: "ada@example.com", active: false}, "stored-hash")
			},
			want: core.ErrInactiveIdentity,
		},
		{
			name:   "missing",
			userID: "missing",
			setup:  func(*fakeIdentityStore) {},
			want:   core.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			identities := newFakeIdentityStore()
			tc.setup(identities)
			refresh := newFakeRefreshStore()
			svc := newTestAuthServiceWithRefresh(t, identities, refresh, &fakeValidationPort{})

			access, refreshToken, err := svc.EstablishSession(context.Background(), tc.userID)
			if !errors.Is(err, tc.want) {
				t.Fatalf("EstablishSession() error = %v, want %v", err, tc.want)
			}
			if access != "" || refreshToken != "" {
				t.Fatal("EstablishSession() returned tokens for unavailable identity")
			}
			if refresh.createN != 0 {
				t.Fatalf("Create calls = %d, want 0", refresh.createN)
			}
		})
	}
}

func TestEstablishSessionCreateFailureReturnsNoTokens(t *testing.T) {
	t.Parallel()

	identities := newFakeIdentityStore()
	identities.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	refresh := newFakeRefreshStore()
	refresh.errCreate = core.ErrUnavailable
	svc := newTestAuthServiceWithRefresh(t, identities, refresh, &fakeValidationPort{})

	access, refreshToken, err := svc.EstablishSession(context.Background(), "user-1")
	if !errors.Is(err, core.ErrUnavailable) {
		t.Fatalf("EstablishSession() error = %v, want ErrUnavailable", err)
	}
	if access != "" || refreshToken != "" {
		t.Fatal("EstablishSession() returned tokens when Create failed")
	}
}

func TestRevokeUserSessionsUsesOptionalCapability(t *testing.T) {
	t.Parallel()

	identities := newFakeIdentityStore()
	identities.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	refresh := newFakeRefreshStore()
	svc := newTestAuthServiceWithRefresh(t, identities, refresh, &fakeValidationPort{})

	if err := svc.RevokeUserSessions(context.Background(), "user-1"); err != nil {
		t.Fatalf("RevokeUserSessions() error = %v", err)
	}
	if refresh.revokeUserN != 1 {
		t.Fatalf("RevokeUser calls = %d, want 1", refresh.revokeUserN)
	}
}

func TestRevokeUserSessionsRejectsUnsupportedStore(t *testing.T) {
	t.Parallel()

	identities := newFakeIdentityStore()
	refresh := newFakeRefreshStore()
	svc, err := NewAuthenticationService[fakeUser](
		identities,
		identities,
		baselineRefreshStore{store: refresh},
		&fakeValidationPort{},
		testJwtConfig(),
	)
	if err != nil {
		t.Fatalf("NewAuthenticationService() error = %v", err)
	}

	err = svc.RevokeUserSessions(context.Background(), "user-1")
	if !errors.Is(err, core.ErrUserSessionRevocationUnsupported) {
		t.Fatalf("RevokeUserSessions() error = %v, want unsupported sentinel", err)
	}
}
