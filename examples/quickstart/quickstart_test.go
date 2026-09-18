package main

import (
	"context"
	"strings"
	"testing"

	"github.com/WilsonSayago/authBase/v3/core"
	"github.com/WilsonSayago/authBase/v3/core/domain"
	"github.com/WilsonSayago/authBase/v3/core/services"
	"github.com/WilsonSayago/authBase/v3/infra/config/properties"
	"github.com/WilsonSayago/authBase/v3/infra/secundary"
)

func TestQuickstartLoginRefreshAuthorize(t *testing.T) {
	t.Parallel()

	cfg := properties.Jwt{
		SecretKey:        strings.Repeat("A", properties.MinSecretBytes),
		RefreshSecret:    strings.Repeat("B", properties.MinSecretBytes),
		ExpirationTime:   1,
		RefreshTokenTime: 24,
		Issuer:           "authbase-example",
		Audience:         "authbase-api",
		LeewaySeconds:    0,
	}
	identity := newMemoryIdentity()
	validation := secundary.NewValidationService()
	hash, err := validation.HashPassword("secret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	identity.add(
		domain.NewUserGeneric("user-1", "Ada", "ada@example.com", nil, false, true),
		"ada@example.com",
		hash,
	)
	refreshStore := newMemoryRefreshStore()

	tokens, err := services.NewTokenManager(cfg)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	authn, err := services.NewAuthenticationService[domain.UserGeneric](
		identity, identity, refreshStore, validation, cfg,
	)
	if err != nil {
		t.Fatalf("NewAuthenticationService() error = %v", err)
	}
	authz, err := services.NewAuthorization[domain.UserGeneric, *demoContext](identity, tokens)
	if err != nil {
		t.Fatalf("NewAuthorization() error = %v", err)
	}

	access, refresh, err := authn.Login(context.Background(), "ada@example.com", "secret")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatal("Login() returned empty tokens")
	}

	httpCtx := &demoContext{header: "Bearer " + access, req: context.Background()}
	authz.AuthorizeJWT()(httpCtx)
	if httpCtx.aborted {
		t.Fatalf("AuthorizeJWT aborted: %d %#v", httpCtx.abortCode, httpCtx.abortBody)
	}

	newAccess, newRefresh, err := authn.RefreshToken(context.Background(), refresh)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if newAccess == "" || newRefresh == "" || newRefresh == refresh {
		t.Fatal("RefreshToken() did not rotate refresh token")
	}

	_, _, err = authn.RefreshToken(context.Background(), refresh)
	if err != core.ErrInvalidRefresh {
		t.Fatalf("replay RefreshToken() error = %v, want ErrInvalidRefresh", err)
	}
}
