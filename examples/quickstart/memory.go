package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/WilsonSayago/authBase/v4/core"
	"github.com/WilsonSayago/authBase/v4/core/domain"
	"github.com/WilsonSayago/authBase/v4/core/port"
)

// memoryIdentity is a demo UserReader + CredentialReader. Not for production.
type memoryIdentity struct {
	mu    sync.Mutex
	users map[string]domain.UserGeneric
	creds map[string]port.CredentialRecord
}

func newMemoryIdentity() *memoryIdentity {
	return &memoryIdentity{
		users: make(map[string]domain.UserGeneric),
		creds: make(map[string]port.CredentialRecord),
	}
}

func (m *memoryIdentity) add(user domain.UserGeneric, username, passwordHash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.GetId()] = user
	m.creds[username] = port.CredentialRecord{
		UserID:       user.GetId(),
		Username:     username,
		PasswordHash: passwordHash,
		Active:       user.GetActive(),
	}
}

func (m *memoryIdentity) FindByID(ctx context.Context, id string) (domain.UserGeneric, error) {
	if err := ctx.Err(); err != nil {
		return domain.UserGeneric{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[id]
	if !ok {
		return domain.UserGeneric{}, core.ErrNotFound
	}
	return user, nil
}

func (m *memoryIdentity) FindCredentialsByUsername(ctx context.Context, username string) (port.CredentialRecord, error) {
	if err := ctx.Err(); err != nil {
		return port.CredentialRecord{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cred, ok := m.creds[username]
	if !ok {
		return port.CredentialRecord{}, core.ErrNotFound
	}
	return cred, nil
}

// memoryRefreshStore is a demo RefreshTokenStore with atomic Rotate semantics.
type memoryRefreshStore struct {
	mu              sync.Mutex
	byHash          map[[32]byte]domain.RefreshSession
	consumed        map[[32]byte]struct{}
	revokedFamilies map[string]struct{}
	now             func() time.Time
}

func newMemoryRefreshStore() *memoryRefreshStore {
	return &memoryRefreshStore{
		byHash:          make(map[[32]byte]domain.RefreshSession),
		consumed:        make(map[[32]byte]struct{}),
		revokedFamilies: make(map[string]struct{}),
		now:             time.Now,
	}
}

func (s *memoryRefreshStore) Create(ctx context.Context, session domain.RefreshSession) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !session.Valid() {
		return fmt.Errorf("invalid refresh session")
	}
	if _, exists := s.byHash[session.TokenHash]; exists {
		return fmt.Errorf("refresh session already exists")
	}
	if _, revoked := s.revokedFamilies[session.FamilyID]; revoked {
		return core.ErrRefreshFamilyRevoked
	}
	s.byHash[session.TokenHash] = session
	return nil
}

func (s *memoryRefreshStore) Rotate(ctx context.Context, currentHash [32]byte, next domain.RefreshSession) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !next.Valid() {
		return fmt.Errorf("invalid refresh session")
	}
	current, ok := s.byHash[currentHash]
	if !ok {
		return core.ErrRefreshNotFound
	}
	if _, revoked := s.revokedFamilies[current.FamilyID]; revoked {
		return core.ErrRefreshFamilyRevoked
	}
	if _, consumed := s.consumed[currentHash]; consumed {
		return core.ErrRefreshConsumed
	}
	if s.now().After(current.ExpiresAt) {
		return core.ErrRefreshNotFound
	}
	if next.FamilyID != current.FamilyID {
		return fmt.Errorf("refresh family mismatch")
	}
	if _, exists := s.byHash[next.TokenHash]; exists {
		return fmt.Errorf("next refresh session already exists")
	}
	s.consumed[currentHash] = struct{}{}
	s.byHash[next.TokenHash] = next
	return nil
}

func (s *memoryRefreshStore) RevokeFamily(ctx context.Context, familyID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revokedFamilies[familyID] = struct{}{}
	return nil
}

var (
	_ port.UserReader[domain.UserGeneric] = (*memoryIdentity)(nil)
	_ port.CredentialReader               = (*memoryIdentity)(nil)
	_ port.RefreshTokenStore              = (*memoryRefreshStore)(nil)
)
