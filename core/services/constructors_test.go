package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/WilsonSayago/authBase/core"
	"github.com/WilsonSayago/authBase/core/domain"
	"github.com/WilsonSayago/authBase/core/port"
	"github.com/WilsonSayago/authBase/infra/config/properties"
)

type otherFakeUser struct {
	id     string
	active bool
}

func (u otherFakeUser) GetId() string                       { return u.id }
func (u otherFakeUser) GetEmail() string                    { return u.id + "@example.com" }
func (u otherFakeUser) GetActive() bool                     { return u.active }
func (u otherFakeUser) GetPermissions() []domain.Permission { return nil }
func (u otherFakeUser) GetIsAdmin() bool                    { return false }
func (u otherFakeUser) HasPermission(string, domain.OperationEnum) bool {
	return false
}

type otherIdentityStore struct{}

func (otherIdentityStore) FindCredentialsByUsername(ctx context.Context, username string) (port.CredentialRecord, error) {
	if err := ctx.Err(); err != nil {
		return port.CredentialRecord{}, err
	}
	return port.CredentialRecord{UserID: "other", Username: username, PasswordHash: "x", Active: true}, nil
}

func (otherIdentityStore) FindByID(ctx context.Context, id string) (otherFakeUser, error) {
	if err := ctx.Err(); err != nil {
		return otherFakeUser{}, err
	}
	return otherFakeUser{id: id, active: true}, nil
}

type fakeRolePort struct {
	mu   sync.Mutex
	name string
}

func (p *fakeRolePort) FindById(id string) (domain.Role, error) {
	return domain.Role{Base: domain.Base{Id: id}, Name: p.name}, nil
}

func (p *fakeRolePort) FindAll(int, int) ([]domain.Role, int, error) {
	return nil, 0, nil
}

func (p *fakeRolePort) Save(role domain.Role) (domain.Role, error) {
	return role, nil
}

func (p *fakeRolePort) Update(domain.Role) error { return nil }

func (p *fakeRolePort) ChangeStatus(string) error { return nil }

type stubContext struct{}

func (stubContext) GetHeader(string) string              { return "" }
func (stubContext) AbortWithStatusJSON(int, interface{}) {}
func (stubContext) Set(string, interface{})              {}
func (stubContext) Next()                                {}
func (stubContext) Status(int)                           {}
func (stubContext) Get(string) (interface{}, bool)       { return nil, false }
func (stubContext) RequestContext() context.Context      { return context.Background() }

func constructorJwt(suffix string) properties.Jwt {
	return properties.Jwt{
		SecretKey:        strings.Repeat("A", properties.MinSecretBytes-len(suffix)) + suffix,
		RefreshSecret:    strings.Repeat("B", properties.MinSecretBytes-len(suffix)) + suffix,
		ExpirationTime:   1,
		RefreshTokenTime: 24,
		Issuer:           "authbase-" + suffix,
		Audience:         "clients-" + suffix,
		LeewaySeconds:    0,
	}
}

func TestGetAuthenticationInstanceReturnsIndependentPointers(t *testing.T) {
	t.Parallel()

	storeA := newFakeIdentityStore()
	storeB := newFakeIdentityStore()
	validateA := &fakeValidationPort{}
	validateB := &fakeValidationPort{}
	cfgA := constructorJwt("aaa")
	cfgB := constructorJwt("bbb")

	first, err := NewAuthenticationService[fakeUser](storeA, storeA, validateA, cfgA)
	if err != nil {
		t.Fatalf("NewAuthenticationService(A) error = %v", err)
	}
	second, err := NewAuthenticationService[fakeUser](storeB, storeB, validateB, cfgB)
	if err != nil {
		t.Fatalf("NewAuthenticationService(B) error = %v", err)
	}
	if first == second {
		t.Fatal("NewAuthenticationService returned the same pointer twice")
	}
	if first.users != storeA || first.credentials != storeA || first.validatePort != validateA || first.tokens == nil {
		t.Fatal("first instance did not keep its own dependencies")
	}
	if second.users != storeB || second.credentials != storeB || second.validatePort != validateB || second.tokens == nil {
		t.Fatal("second instance did not keep its own dependencies")
	}
	if first.tokens == second.tokens {
		t.Fatal("authentication services unexpectedly share TokenManager")
	}
}

func TestGetAuthenticationInstanceDistinctGenericTypesDoNotCollide(t *testing.T) {
	t.Parallel()

	cfg := constructorJwt("shared")
	validate := &fakeValidationPort{}
	store := newFakeIdentityStore()
	other := otherIdentityStore{}

	first, err := NewAuthenticationService[fakeUser](store, store, validate, cfg)
	if err != nil {
		t.Fatalf("NewAuthenticationService(fakeUser) error = %v", err)
	}
	second, err := NewAuthenticationService[otherFakeUser](other, other, validate, cfg)
	if err != nil {
		t.Fatalf("NewAuthenticationService(otherFakeUser) error = %v", err)
	}
	if first == nil || second == nil {
		t.Fatal("expected non-nil services for distinct generic types")
	}
}

func TestGetAuthenticationInstanceConcurrentIndependentDependencies(t *testing.T) {
	t.Parallel()

	const n = 100
	results := make([]*AuthenticationService[fakeUser], n)
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			store := newFakeIdentityStore()
			validate := &fakeValidationPort{}
			cfg := constructorJwt(fmt.Sprintf("%03d", i))
			got, err := NewAuthenticationService[fakeUser](store, store, validate, cfg)
			if err != nil {
				t.Errorf("instance %d error = %v", i, err)
				return
			}
			if got.users != store || got.credentials != store || got.validatePort != validate || got.tokens == nil {
				t.Errorf("instance %d kept another call's dependencies", i)
				return
			}
			results[i] = got
		}()
	}
	wg.Wait()

	seen := make(map[*AuthenticationService[fakeUser]]struct{}, n)
	for i, svc := range results {
		if svc == nil {
			t.Fatalf("missing instance at index %d", i)
		}
		if _, ok := seen[svc]; ok {
			t.Fatalf("duplicate pointer returned for concurrent constructions")
		}
		seen[svc] = struct{}{}
	}
}

func TestNewAuthorizationReturnsIndependentPointers(t *testing.T) {
	t.Parallel()

	storeA := newFakeIdentityStore()
	storeB := newFakeIdentityStore()
	cfgA := constructorJwt("authza")
	cfgB := constructorJwt("authzb")

	first, err := NewAuthorization[fakeUser, stubContext](storeA, cfgA)
	if err != nil {
		t.Fatalf("NewAuthorization(A) error = %v", err)
	}
	second, err := NewAuthorization[fakeUser, stubContext](storeB, cfgB)
	if err != nil {
		t.Fatalf("NewAuthorization(B) error = %v", err)
	}
	if first == second {
		t.Fatal("NewAuthorization returned the same pointer twice")
	}
	if first.users != storeA || first.tokens == nil {
		t.Fatal("first authorization instance did not keep its own dependencies")
	}
	if second.users != storeB || second.tokens == nil {
		t.Fatal("second authorization instance did not keep its own dependencies")
	}
	if first.tokens == second.tokens {
		t.Fatal("authorization services unexpectedly share TokenManager")
	}
}

func TestNewAuthorizationRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	cfg := constructorJwt("bad")
	cfg.Audience = ""
	svc, err := NewAuthorization[fakeUser, stubContext](newFakeIdentityStore(), cfg)
	if err == nil {
		t.Fatal("NewAuthorization() error = nil, want invalid config error")
	}
	if svc != nil {
		t.Fatal("NewAuthorization() returned service for invalid config")
	}
}

func TestGetRoleServiceInstanceReturnsIndependentPointers(t *testing.T) {
	t.Parallel()

	portA := &fakeRolePort{name: "a"}
	portB := &fakeRolePort{name: "b"}

	first := GetRoleServiceInstance(portA)
	second := GetRoleServiceInstance(portB)

	svcA, ok := first.(*RoleService)
	if !ok {
		t.Fatalf("first type = %T, want *RoleService", first)
	}
	svcB, ok := second.(*RoleService)
	if !ok {
		t.Fatalf("second type = %T, want *RoleService", second)
	}
	if svcA == svcB {
		t.Fatal("GetRoleServiceInstance returned the same pointer twice")
	}
	if svcA.port != portA || svcB.port != portB {
		t.Fatal("role service instances did not keep their own ports")
	}
}

var _ core.Context = stubContext{}
