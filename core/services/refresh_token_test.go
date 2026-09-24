package services

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WilsonSayago/authBase/v4/core"
	"github.com/WilsonSayago/authBase/v4/core/domain"
)

func TestRefreshTokenRotatesAndPreservesFamily(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	refreshStore := newFakeRefreshStore()
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthServiceWithRefresh(t, store, refreshStore, validate)

	access, refresh, err := svc.Login(context.Background(), "ada@example.com", "plain-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if refreshStore.createN != 1 {
		t.Fatalf("Create calls = %d, want 1", refreshStore.createN)
	}

	firstClaims, err := svc.tokens.ParseRefresh(refresh)
	if err != nil {
		t.Fatalf("ParseRefresh() error = %v", err)
	}

	newAccess, newRefresh, err := svc.RefreshToken(context.Background(), refresh)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if newAccess == "" || newRefresh == "" || newRefresh == refresh {
		t.Fatal("RefreshToken() must return a new refresh token")
	}
	if access == "" {
		t.Fatal("login access token unexpectedly empty")
	}

	secondClaims, err := svc.tokens.ParseRefresh(newRefresh)
	if err != nil {
		t.Fatalf("ParseRefresh(new) error = %v", err)
	}
	if secondClaims.FamilyID != firstClaims.FamilyID {
		t.Fatalf("family = %q, want %q", secondClaims.FamilyID, firstClaims.FamilyID)
	}
	if secondClaims.ID == firstClaims.ID {
		t.Fatal("rotated refresh must use a new jti")
	}
	if domain.HashRefreshToken(newRefresh) == domain.HashRefreshToken(refresh) {
		t.Fatal("rotated refresh must use a new hash")
	}
	if refreshStore.rotateN != 1 {
		t.Fatalf("Rotate calls = %d, want 1", refreshStore.rotateN)
	}
}

func TestRefreshTokenReplayRevokesFamily(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	refreshStore := newFakeRefreshStore()
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthServiceWithRefresh(t, store, refreshStore, validate)

	_, refresh, err := svc.Login(context.Background(), "ada@example.com", "plain-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	claims, err := svc.tokens.ParseRefresh(refresh)
	if err != nil {
		t.Fatalf("ParseRefresh() error = %v", err)
	}

	_, nextRefresh, err := svc.RefreshToken(context.Background(), refresh)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}

	access, replayRefresh, err := svc.RefreshToken(context.Background(), refresh)
	if !errors.Is(err, core.ErrInvalidRefresh) {
		t.Fatalf("replay error = %v, want ErrInvalidRefresh", err)
	}
	if access != "" || replayRefresh != "" {
		t.Fatal("replay must not return tokens")
	}
	if refreshStore.revokeN != 1 {
		t.Fatalf("RevokeFamily calls = %d, want 1", refreshStore.revokeN)
	}

	access, replayRefresh, err = svc.RefreshToken(context.Background(), nextRefresh)
	if !errors.Is(err, core.ErrInvalidRefresh) {
		t.Fatalf("post-revoke refresh error = %v, want ErrInvalidRefresh", err)
	}
	if access != "" || replayRefresh != "" {
		t.Fatal("revoked family must not rotate")
	}

	_ = claims
}

func TestTokenFamilyLoginPersistsSession(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	refreshStore := newFakeRefreshStore()
	refreshStore.errCreate = core.ErrUnavailable
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthServiceWithRefresh(t, store, refreshStore, validate)

	access, refresh, err := svc.Login(context.Background(), "ada@example.com", "plain-password")
	if !errors.Is(err, core.ErrUnavailable) {
		t.Fatalf("Login() error = %v, want ErrUnavailable", err)
	}
	if access != "" || refresh != "" {
		t.Fatal("Login() must not return tokens when store Create fails")
	}
}

func TestRefreshTokenActiveUserRequired(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	refreshStore := newFakeRefreshStore()
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthServiceWithRefresh(t, store, refreshStore, validate)

	_, refresh, err := svc.Login(context.Background(), "ada@example.com", "plain-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	store.setActive("user-1", false)
	access, newRefresh, err := svc.RefreshToken(context.Background(), refresh)
	if !errors.Is(err, core.ErrInvalidRefresh) {
		t.Fatalf("RefreshToken() error = %v, want ErrInvalidRefresh", err)
	}
	if access != "" || newRefresh != "" {
		t.Fatal("RefreshToken() returned tokens for inactive identity")
	}
}

func TestRefreshTokenRespectsCanceledContext(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	refreshStore := newFakeRefreshStore()
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthServiceWithRefresh(t, store, refreshStore, validate)

	_, refresh, err := svc.Login(context.Background(), "ada@example.com", "plain-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	access, newRefresh, err := svc.RefreshToken(ctx, refresh)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RefreshToken() error = %v, want context.Canceled", err)
	}
	if access != "" || newRefresh != "" {
		t.Fatal("canceled refresh must not return tokens")
	}
}

func TestConcurrentRefresh(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	refreshStore := newFakeRefreshStore()
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthServiceWithRefresh(t, store, refreshStore, validate)

	_, refresh, err := svc.Login(context.Background(), "ada@example.com", "plain-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	var success atomic.Int64
	var failures atomic.Int64
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			access, newRefresh, err := svc.RefreshToken(context.Background(), refresh)
			if err != nil {
				failures.Add(1)
				if !errors.Is(err, core.ErrInvalidRefresh) {
					t.Errorf("unexpected error = %v", err)
				}
				if access != "" || newRefresh != "" {
					t.Error("failed refresh returned tokens")
				}
				return
			}
			success.Add(1)
			if access == "" || newRefresh == "" {
				t.Error("successful refresh returned empty tokens")
			}
		}()
	}
	wg.Wait()

	if success.Load() != 1 || failures.Load() != 1 {
		t.Fatalf("concurrent refresh success=%d failures=%d, want 1/1", success.Load(), failures.Load())
	}
}

func TestFakeRefreshStoreRotateInvariants(t *testing.T) {
	t.Parallel()

	store := newFakeRefreshStore()
	session := domain.RefreshSession{
		TokenID:   "jti-1",
		FamilyID:  "family-1",
		UserID:    "user-1",
		TokenHash: domain.HashRefreshToken("token-1"),
		IssuedAt:  store.now(),
		ExpiresAt: store.now().Add(time.Hour),
	}
	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	next := session
	next.TokenID = "jti-2"
	next.TokenHash = domain.HashRefreshToken("token-2")
	if err := store.Rotate(context.Background(), session.TokenHash, next); err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	if err := store.Rotate(context.Background(), session.TokenHash, next); !errors.Is(err, core.ErrRefreshConsumed) {
		t.Fatalf("second Rotate() error = %v, want ErrRefreshConsumed", err)
	}
}
