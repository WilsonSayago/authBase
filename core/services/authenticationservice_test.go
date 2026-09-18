package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/WilsonSayago/authBase/infra/config/properties"
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

func newTestAuthService(t *testing.T, port *fakeGenericPort, validate *fakeValidationPort) *AuthenticationService[fakeUser] {
	t.Helper()
	svc, err := NewAuthenticationService[fakeUser](port, validate, testJwtConfig())
	if err != nil {
		t.Fatalf("NewAuthenticationService() error = %v", err)
	}
	return svc
}

func TestAuthenticationLoginValidReturnsTokens(t *testing.T) {
	t.Parallel()

	port := newFakeGenericPort()
	port.add(fakeUser{
		id:       "user-1",
		email:    "ada@example.com",
		password: "stored-hash",
	})
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthService(t, port, validate)

	access, refresh, err := svc.Login("ada@example.com", "plain-password")
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

func TestAuthenticationLoginInvalidPassword(t *testing.T) {
	t.Parallel()

	port := newFakeGenericPort()
	port.add(fakeUser{
		id:       "user-1",
		email:    "ada@example.com",
		password: "stored-hash",
	})
	validate := &fakeValidationPort{
		checkPassword: func(string, string) bool { return false },
	}
	svc := newTestAuthService(t, port, validate)

	access, refresh, err := svc.Login("ada@example.com", "wrong-password")
	if err == nil {
		t.Fatal("Login() error = nil, want invalid password error")
	}
	if access != "" || refresh != "" {
		t.Fatal("Login() returned tokens for invalid password")
	}
}

func TestAuthenticationLoginPropagatesFindByEmailError(t *testing.T) {
	t.Parallel()

	port := newFakeGenericPort()
	port.errByEmail = errors.New("lookup failed")
	validate := &fakeValidationPort{}
	svc := newTestAuthService(t, port, validate)

	access, refresh, err := svc.Login("missing@example.com", "any")
	if err == nil {
		t.Fatal("Login() error = nil, want FindByEmail error")
	}
	if access != "" || refresh != "" {
		t.Fatal("Login() returned tokens when FindByEmail failed")
	}
	if validate.checkCalls != 0 {
		t.Fatalf("CheckPassword calls = %d, want 0", validate.checkCalls)
	}
}

func TestAuthenticationValidateTokenReturnsUser(t *testing.T) {
	t.Parallel()

	port := newFakeGenericPort()
	user := fakeUser{
		id:       "user-1",
		email:    "ada@example.com",
		password: "stored-hash",
	}
	port.add(user)
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthService(t, port, validate)

	access, _, err := svc.Login("ada@example.com", "plain-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	got, err := svc.ValidateToken(access)
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

	port := newFakeGenericPort()
	port.add(fakeUser{
		id:       "user-1",
		email:    "ada@example.com",
		password: "stored-hash",
	})
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	svc := newTestAuthService(t, port, validate)

	access, _, err := svc.Login("ada@example.com", "plain-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	foreignCfg := testJwtConfig()
	foreignCfg.SecretKey = strings.Repeat("C", properties.MinSecretBytes)
	foreignCfg.RefreshSecret = strings.Repeat("D", properties.MinSecretBytes)
	foreign, err := NewAuthenticationService[fakeUser](port, validate, foreignCfg)
	if err != nil {
		t.Fatalf("NewAuthenticationService(foreign) error = %v", err)
	}

	got, err := foreign.ValidateToken(access)
	if err == nil {
		t.Fatal("ValidateToken() error = nil, want rejection for foreign signing key")
	}
	if got != nil {
		t.Fatal("ValidateToken() returned user for foreign signing key")
	}
}

func TestNewAuthenticationServiceRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	cfg := testJwtConfig()
	cfg.Issuer = ""
	svc, err := NewAuthenticationService[fakeUser](newFakeGenericPort(), &fakeValidationPort{}, cfg)
	if err == nil {
		t.Fatal("NewAuthenticationService() error = nil, want invalid config error")
	}
	if svc != nil {
		t.Fatal("NewAuthenticationService() returned service for invalid config")
	}
}

func TestGetAuthenticationInstanceRejectsNilConfig(t *testing.T) {
	t.Parallel()

	svc, err := GetAuthenticationInstance[fakeUser](newFakeGenericPort(), &fakeValidationPort{}, nil)
	if err == nil {
		t.Fatal("GetAuthenticationInstance() error = nil, want nil config error")
	}
	if svc != nil {
		t.Fatal("GetAuthenticationInstance() returned service for nil config")
	}
}
