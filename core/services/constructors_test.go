package services

import (
	"fmt"
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

func TestGetAuthenticationInstanceReturnsIndependentPointers(t *testing.T) {
	t.Parallel()

	portA := newFakeGenericPort()
	portB := newFakeGenericPort()
	validateA := &fakeValidationPort{}
	validateB := &fakeValidationPort{}
	propA := &properties.JwtProp{Jwt: properties.Jwt{SecretKey: "secret-a", RefreshSecret: "refresh-a", ExpirationTime: 1, RefreshTokenTime: 2}}
	propB := &properties.JwtProp{Jwt: properties.Jwt{SecretKey: "secret-b", RefreshSecret: "refresh-b", ExpirationTime: 3, RefreshTokenTime: 4}}

	first := GetAuthenticationInstance[fakeUser](portA, validateA, propA)
	second := GetAuthenticationInstance[fakeUser](portB, validateB, propB)

	svcA, ok := first.(*AuthenticationService[fakeUser])
	if !ok {
		t.Fatalf("first type = %T, want *AuthenticationService[fakeUser]", first)
	}
	svcB, ok := second.(*AuthenticationService[fakeUser])
	if !ok {
		t.Fatalf("second type = %T, want *AuthenticationService[fakeUser]", second)
	}
	if svcA == svcB {
		t.Fatal("GetAuthenticationInstance returned the same pointer twice")
	}
	if svcA.port != portA || svcA.validatePort != validateA || svcA.prop != propA {
		t.Fatal("first instance did not keep its own dependencies")
	}
	if svcB.port != portB || svcB.validatePort != validateB || svcB.prop != propB {
		t.Fatal("second instance did not keep its own dependencies")
	}
}

func TestGetAuthenticationInstanceDistinctGenericTypesDoNotCollide(t *testing.T) {
	t.Parallel()

	prop := &properties.JwtProp{Jwt: properties.Jwt{SecretKey: "secret", RefreshSecret: "refresh", ExpirationTime: 1, RefreshTokenTime: 2}}
	validate := &fakeValidationPort{}

	first := GetAuthenticationInstance[fakeUser](newFakeGenericPort(), validate, prop)
	second := GetAuthenticationInstance[otherFakeUser](otherFakeGenericPort{}, validate, prop)

	if _, ok := first.(*AuthenticationService[fakeUser]); !ok {
		t.Fatalf("first type = %T, want *AuthenticationService[fakeUser]", first)
	}
	if _, ok := second.(*AuthenticationService[otherFakeUser]); !ok {
		t.Fatalf("second type = %T, want *AuthenticationService[otherFakeUser]", second)
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
			prop := &properties.JwtProp{Jwt: properties.Jwt{
				SecretKey:        fmt.Sprintf("secret-%d", i),
				RefreshSecret:    fmt.Sprintf("refresh-%d", i),
				ExpirationTime:   1,
				RefreshTokenTime: 2,
			}}
			got := GetAuthenticationInstance[fakeUser](port, validate, prop)
			svc, ok := got.(*AuthenticationService[fakeUser])
			if !ok {
				t.Errorf("instance %d type = %T", i, got)
				return
			}
			if svc.port != port || svc.validatePort != validate || svc.prop != prop {
				t.Errorf("instance %d kept another call's dependencies", i)
				return
			}
			results[i] = svc
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
	propA := &properties.JwtProp{Jwt: properties.Jwt{SecretKey: "authz-a"}}
	propB := &properties.JwtProp{Jwt: properties.Jwt{SecretKey: "authz-b"}}

	first := NewAuthorization[fakeUser, stubContext](portA, propA)
	second := NewAuthorization[fakeUser, stubContext](portB, propB)

	svcA, ok := first.(*Authorization[fakeUser, stubContext])
	if !ok {
		t.Fatalf("first type = %T, want *Authorization[fakeUser, stubContext]", first)
	}
	svcB, ok := second.(*Authorization[fakeUser, stubContext])
	if !ok {
		t.Fatalf("second type = %T, want *Authorization[fakeUser, stubContext]", second)
	}
	if svcA == svcB {
		t.Fatal("NewAuthorization returned the same pointer twice")
	}
	if svcA.port != portA || svcA.prop != propA {
		t.Fatal("first authorization instance did not keep its own dependencies")
	}
	if svcB.port != portB || svcB.prop != propB {
		t.Fatal("second authorization instance did not keep its own dependencies")
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
