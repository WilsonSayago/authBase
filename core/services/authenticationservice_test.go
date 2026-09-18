package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/WilsonSayago/authBase/v3/core"
	"github.com/WilsonSayago/authBase/v3/core/domain"
	"github.com/WilsonSayago/authBase/v3/infra/config/properties"
)

func testJwtConfig() properties.Jwt {
	return properties.Jwt{
		SecretKey:        strings.Repeat("A", properties.MinSecretBytes),
		RefreshSecret:    strings.Repeat("B", properties.MinSecretBytes),
		ExpirationTime:   1,
		RefreshTokenTime: 24,
		Issuer:           "authbase-test",
		Audience:         "authbase-clients",
		LeewaySeconds:    0,
	}
}

func newTestAuthService(t *testing.T, store *fakeIdentityStore, validate *fakeValidationPort) *AuthenticationService[fakeUser] {
	t.Helper()
	return newTestAuthServiceWithRefresh(t, store, newFakeRefreshStore(), validate)
}

func newTestAuthServiceWithRefresh(
	t *testing.T,
	store *fakeIdentityStore,
	refresh *fakeRefreshStore,
	validate *fakeValidationPort,
) *AuthenticationService[fakeUser] {
	t.Helper()
	svc, err := NewAuthenticationService[fakeUser](store, store, refresh, validate, testJwtConfig())
	if err != nil {
		t.Fatalf("NewAuthenticationService() error = %v", err)
	}
	return svc
}

func TestAuthenticationLoginValidReturnsTokens(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthService(t, store, validate)

	access, refresh, err := svc.Login(context.Background(), "ada@example.com", "plain-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatalf("Login() returned empty tokens: access empty=%v refresh empty=%v", access == "", refresh == "")
	}
	if validate.checkCalls != 1 {
		t.Fatalf("CheckPassword calls = %d, want 1", validate.checkCalls)
	}
}

func TestInvalidCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		username string
		password string
		setup    func(*fakeIdentityStore, *fakeValidationPort)
	}{
		{
			name:     "wrong password",
			username: "ada@example.com",
			password: "wrong",
			setup: func(_ *fakeIdentityStore, validate *fakeValidationPort) {
				validate.checkPassword = func(string, string) bool { return false }
			},
		},
		{
			name:     "missing user",
			username: "missing@example.com",
			password: "any",
		},
		{
			name:     "inactive user",
			username: "ada@example.com",
			password: "plain-password",
			setup: func(store *fakeIdentityStore, _ *fakeValidationPort) {
				store.setActive("user-1", false)
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := newFakeIdentityStore()
			store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
			validate := &fakeValidationPort{
				checkPassword: func(hashedPassword, password string) bool {
					return hashedPassword == "stored-hash" && password == "plain-password"
				},
			}
			if tc.setup != nil {
				tc.setup(store, validate)
			}
			svc := newTestAuthService(t, store, validate)

			access, refresh, err := svc.Login(context.Background(), tc.username, tc.password)
			if !errors.Is(err, core.ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
			}
			if access != "" || refresh != "" {
				t.Fatal("Login() returned tokens for invalid credentials")
			}
		})
	}
}

func TestAuthenticationLoginPropagatesInfrastructureError(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.errByUsername = core.ErrUnavailable
	validate := &fakeValidationPort{}
	svc := newTestAuthService(t, store, validate)

	access, refresh, err := svc.Login(context.Background(), "ada@example.com", "any")
	if !errors.Is(err, core.ErrUnavailable) {
		t.Fatalf("Login() error = %v, want ErrUnavailable", err)
	}
	if access != "" || refresh != "" {
		t.Fatal("Login() returned tokens when store failed")
	}
	if validate.checkCalls != 0 {
		t.Fatalf("CheckPassword calls = %d, want 0", validate.checkCalls)
	}
}

func TestAuthenticationLoginRespectsCanceledContext(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	svc := newTestAuthService(t, store, &fakeValidationPort{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := svc.Login(ctx, "ada@example.com", "plain-password")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Login() error = %v, want context.Canceled", err)
	}
}

func TestAuthenticationValidateTokenReturnsUser(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	user := fakeUser{id: "user-1", email: "ada@example.com", active: true}
	store.add(user, "stored-hash")
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthService(t, store, validate)

	access, _, err := svc.Login(context.Background(), "ada@example.com", "plain-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	got, err := svc.ValidateToken(context.Background(), access)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if got.GetId() != user.id {
		t.Fatalf("ValidateToken() id = %q, want %q", got.GetId(), user.id)
	}
	if got.GetEmail() != user.email {
		t.Fatalf("ValidateToken() email = %q, want %q", got.GetEmail(), user.email)
	}
}

func TestAuthenticationValidateTokenRejectsForeignKey(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthService(t, store, validate)

	access, _, err := svc.Login(context.Background(), "ada@example.com", "plain-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	foreignCfg := testJwtConfig()
	foreignCfg.SecretKey = strings.Repeat("C", properties.MinSecretBytes)
	foreignCfg.RefreshSecret = strings.Repeat("D", properties.MinSecretBytes)
	foreign, err := NewAuthenticationService[fakeUser](store, store, newFakeRefreshStore(), validate, foreignCfg)
	if err != nil {
		t.Fatalf("NewAuthenticationService(foreign) error = %v", err)
	}

	got, err := foreign.ValidateToken(context.Background(), access)
	if err == nil {
		t.Fatal("ValidateToken() error = nil, want rejection for foreign signing key")
	}
	if got != nil {
		t.Fatal("ValidateToken() returned user for foreign signing key")
	}
}

func TestValidateTokenActiveUserRequired(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthService(t, store, validate)

	access, _, err := svc.Login(context.Background(), "ada@example.com", "plain-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	store.setActive("user-1", false)
	got, err := svc.ValidateToken(context.Background(), access)
	if !errors.Is(err, core.ErrInactiveIdentity) {
		t.Fatalf("ValidateToken() error = %v, want ErrInactiveIdentity", err)
	}
	if got != nil {
		t.Fatal("ValidateToken() returned user for inactive identity")
	}
}

func TestNewAuthenticationServiceRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	cfg := testJwtConfig()
	cfg.Issuer = ""
	store := newFakeIdentityStore()
	svc, err := NewAuthenticationService[fakeUser](store, store, newFakeRefreshStore(), &fakeValidationPort{}, cfg)
	if err == nil {
		t.Fatal("NewAuthenticationService() error = nil, want invalid config error")
	}
	if svc != nil {
		t.Fatal("NewAuthenticationService() returned service for invalid config")
	}
}

func TestGetAuthenticationInstanceRejectsNilConfig(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	svc, err := GetAuthenticationInstance[fakeUser](store, store, newFakeRefreshStore(), &fakeValidationPort{}, nil)
	if err == nil {
		t.Fatal("GetAuthenticationInstance() error = nil, want nil config error")
	}
	if svc != nil {
		t.Fatal("GetAuthenticationInstance() returned service for nil config")
	}
}

func TestIsAuthorizedRejectsInactiveAdmin(t *testing.T) {
	t.Parallel()

	authz := &Authorization[fakeUser, stubContext]{}
	user := fakeUser{id: "admin-1", isAdmin: true, active: false}
	if authz.IsAuthorized(user, nil, "users", domain.READ) {
		t.Fatal("inactive admin must not be authorized")
	}
}
