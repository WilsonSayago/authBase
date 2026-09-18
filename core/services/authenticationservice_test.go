package services

import (
	"errors"
	"testing"

	"github.com/WilsonSayago/authBase/infra/config/properties"
	"github.com/golang-jwt/jwt/v5"
)

func newTestAuthService(port *fakeGenericPort, validate *fakeValidationPort) *AuthenticationService[fakeUser] {
	return &AuthenticationService[fakeUser]{
		port:         port,
		validatePort: validate,
		prop: &properties.JwtProp{
			Jwt: properties.Jwt{
				SecretKey:        "test-access-secret",
				RefreshSecret:    "test-refresh-secret",
				ExpirationTime:   1,
				RefreshTokenTime: 2,
			},
		},
	}
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
	svc := newTestAuthService(port, validate)

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
	svc := newTestAuthService(port, validate)

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
	svc := newTestAuthService(port, validate)

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
	svc := newTestAuthService(port, validate)

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
	svc := newTestAuthService(port, validate)

	access, _, err := svc.Login("ada@example.com", "plain-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	foreign := &AuthenticationService[fakeUser]{
		port:         port,
		validatePort: validate,
		prop: &properties.JwtProp{
			Jwt: properties.Jwt{
				SecretKey:        "other-access-secret",
				RefreshSecret:    "other-refresh-secret",
				ExpirationTime:   1,
				RefreshTokenTime: 2,
			},
		},
	}

	got, err := foreign.ValidateToken(access)
	if err == nil {
		t.Fatal("ValidateToken() error = nil, want rejection for foreign signing key")
	}
	if got != nil {
		t.Fatal("ValidateToken() returned user for foreign signing key")
	}

	// Ensure the original token is still a well-formed JWT so the failure is key-related.
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	if _, _, parseErr := parser.ParseUnverified(access, jwt.MapClaims{}); parseErr != nil {
		t.Fatalf("access token is not a JWT: %v", parseErr)
	}
}
