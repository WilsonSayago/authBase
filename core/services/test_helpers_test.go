package services

import (
	"errors"
	"sync"

	domain "github.com/WilsonSayago/authBase/core/domain"
)

type fakeUser struct {
	id       string
	email    string
	password string
	isAdmin  bool
	active   bool
}

func (u fakeUser) GetId() string {
	return u.id
}

func (u fakeUser) GetEmail() string {
	return u.email
}

func (u fakeUser) GetPassword() string {
	return u.password
}

func (u fakeUser) GetPermissions() []domain.Permission {
	return nil
}

func (u fakeUser) GetIsAdmin() bool {
	return u.isAdmin
}

func (u fakeUser) HasPermission(string, domain.OperationEnum) bool {
	return false
}

type fakeGenericPort struct {
	mu            sync.Mutex
	byEmail       map[string]fakeUser
	byID          map[string]fakeUser
	findByEmailN  int
	findFullByIDN int
	errByEmail    error
	errByID       error
}

func newFakeGenericPort() *fakeGenericPort {
	return &fakeGenericPort{
		byEmail: make(map[string]fakeUser),
		byID:    make(map[string]fakeUser),
	}
}

func (p *fakeGenericPort) add(user fakeUser) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.byEmail[user.email] = user
	p.byID[user.id] = user
}

func (p *fakeGenericPort) FindByEmail(email string) (fakeUser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.findByEmailN++
	if p.errByEmail != nil {
		return fakeUser{}, p.errByEmail
	}
	user, ok := p.byEmail[email]
	if !ok {
		return fakeUser{}, errors.New("user not found")
	}
	return user, nil
}

func (p *fakeGenericPort) FindFullById(id string) (fakeUser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.findFullByIDN++
	if p.errByID != nil {
		return fakeUser{}, p.errByID
	}
	user, ok := p.byID[id]
	if !ok {
		return fakeUser{}, errors.New("user not found")
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
