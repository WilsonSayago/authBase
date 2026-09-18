package services

import (
	"context"
	"sync"

	"github.com/WilsonSayago/authBase/core"
	domain "github.com/WilsonSayago/authBase/core/domain"
	"github.com/WilsonSayago/authBase/core/port"
)

type fakeUser struct {
	id            string
	email         string
	isAdmin       bool
	active        bool
	hasPermission func(entity string, operation domain.OperationEnum) bool
}

func (u fakeUser) GetId() string {
	return u.id
}

func (u fakeUser) GetEmail() string {
	return u.email
}

func (u fakeUser) GetActive() bool {
	return u.active
}

func (u fakeUser) GetPermissions() []domain.Permission {
	return nil
}

func (u fakeUser) GetIsAdmin() bool {
	return u.isAdmin
}

func (u fakeUser) HasPermission(entity string, operation domain.OperationEnum) bool {
	if u.hasPermission != nil {
		return u.hasPermission(entity, operation)
	}
	return false
}

type fakeIdentityStore struct {
	mu            sync.Mutex
	byUsername    map[string]port.CredentialRecord
	byID          map[string]fakeUser
	findCredN     int
	findByIDN     int
	errByUsername error
	errByID       error
}

func newFakeIdentityStore() *fakeIdentityStore {
	return &fakeIdentityStore{
		byUsername: make(map[string]port.CredentialRecord),
		byID:       make(map[string]fakeUser),
	}
}

func (p *fakeIdentityStore) add(user fakeUser, passwordHash string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.byID[user.id] = user
	p.byUsername[user.email] = port.CredentialRecord{
		UserID:       user.id,
		Username:     user.email,
		PasswordHash: passwordHash,
		Active:       user.active,
	}
}

func (p *fakeIdentityStore) setActive(id string, active bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	user, ok := p.byID[id]
	if !ok {
		return
	}
	user.active = active
	p.byID[id] = user
	if cred, ok := p.byUsername[user.email]; ok {
		cred.Active = active
		p.byUsername[user.email] = cred
	}
}

func (p *fakeIdentityStore) FindCredentialsByUsername(ctx context.Context, username string) (port.CredentialRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.findCredN++
	if err := ctx.Err(); err != nil {
		return port.CredentialRecord{}, err
	}
	if p.errByUsername != nil {
		return port.CredentialRecord{}, p.errByUsername
	}
	cred, ok := p.byUsername[username]
	if !ok {
		return port.CredentialRecord{}, core.ErrNotFound
	}
	return cred, nil
}

func (p *fakeIdentityStore) FindByID(ctx context.Context, id string) (fakeUser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.findByIDN++
	if err := ctx.Err(); err != nil {
		return fakeUser{}, err
	}
	if p.errByID != nil {
		return fakeUser{}, p.errByID
	}
	user, ok := p.byID[id]
	if !ok {
		return fakeUser{}, core.ErrNotFound
	}
	return user, nil
}

type fakeValidationPort struct {
	mu            sync.Mutex
	checkPassword func(hashedPassword, password string) bool
	checkCalls    int
	hashCalls     int
}

func (v *fakeValidationPort) HashPassword(password string) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.hashCalls++
	return "hashed:" + password, nil
}

func (v *fakeValidationPort) CheckPassword(hashedPassword, password string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.checkCalls++
	if v.checkPassword != nil {
		return v.checkPassword(hashedPassword, password)
	}
	return hashedPassword == password
}

var (
	_ port.CredentialReader     = (*fakeIdentityStore)(nil)
	_ port.UserReader[fakeUser] = (*fakeIdentityStore)(nil)
)
