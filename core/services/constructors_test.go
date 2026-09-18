package services

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/WilsonSayago/authBase/core"
	"github.com/WilsonSayago/authBase/core/domain"
	"github.com/WilsonSayago/authBase/infra/config/properties"
)

type otherFakeUser struct {
	id string
}

func (u otherFakeUser) GetId() string                       { return u.id }
func (u otherFakeUser) GetEmail() string                    { return u.id + "@example.com" }
func (u otherFakeUser) GetPassword() string                 { return "x" }
func (u otherFakeUser) GetPermissions() []domain.Permission { return nil }
func (u otherFakeUser) GetIsAdmin() bool                    { return false }
func (u otherFakeUser) HasPermission(string, domain.OperationEnum) bool {
	return false
}

type otherFakeGenericPort struct{}

func (p otherFakeGenericPort) FindByEmail(string) (otherFakeUser, error) {
	return otherFakeUser{}, fmt.Errorf("not implemented")
}

func (p otherFakeGenericPort) FindFullById(id string) (otherFakeUser, error) {
	return otherFakeUser{id: id}, nil
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

	portA := newFakeGenericPort()
	portB := newFakeGenericPort()
	validateA := &fakeValidationPort{}
	validateB := &fakeValidationPort{}
	cfgA := constructorJwt("aaa")
	cfgB := constructorJwt("bbb")

	first, err := NewAuthenticationService[fakeUser](portA, validateA, cfgA)
	if err != nil {
		t.Fatalf("NewAuthenticationService(A) error = %v", err)
	}
	second, err := NewAuthenticationService[fakeUser](portB, validateB, cfgB)
	if err != nil {
		t.Fatalf("NewAuthenticationService(B) error = %v", err)
	}
	if first == second {
		t.Fatal("NewAuthenticationService returned the same pointer twice")
	}
	if first.port != portA || first.validatePort != validateA || first.tokens == nil {
		t.Fatal("first instance did not keep its own dependencies")
	}
	if second.port != portB || second.validatePort != validateB || second.tokens == nil {
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

	first, err := NewAuthenticationService[fakeUser](newFakeGenericPort(), validate, cfg)
	if err != nil {
		t.Fatalf("NewAuthenticationService(fakeUser) error = %v", err)
	}
	second, err := NewAuthenticationService[otherFakeUser](otherFakeGenericPort{}, validate, cfg)
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
			port := newFakeGenericPort()
			validate := &fakeValidationPort{}
			cfg := constructorJwt(fmt.Sprintf("%03d", i))
			got, err := NewAuthenticationService[fakeUser](port, validate, cfg)
			if err != nil {
				t.Errorf("instance %d error = %v", i, err)
				return
			}
			if got.port != port || got.validatePort != validate || got.tokens == nil {
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

	portA := newFakeGenericPort()
	portB := newFakeGenericPort()
	cfgA := constructorJwt("authza")
	cfgB := constructorJwt("authzb")

	first, err := NewAuthorization[fakeUser, stubContext](portA, cfgA)
	if err != nil {
		t.Fatalf("NewAuthorization(A) error = %v", err)
	}
	second, err := NewAuthorization[fakeUser, stubContext](portB, cfgB)
	if err != nil {
		t.Fatalf("NewAuthorization(B) error = %v", err)
	}
	if first == second {
		t.Fatal("NewAuthorization returned the same pointer twice")
	}
	if first.port != portA || first.tokens == nil {
		t.Fatal("first authorization instance did not keep its own dependencies")
	}
	if second.port != portB || second.tokens == nil {
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
	svc, err := NewAuthorization[fakeUser, stubContext](newFakeGenericPort(), cfg)
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

// Ensure stubContext satisfies core.Context at compile time.
var _ core.Context = stubContext{}
