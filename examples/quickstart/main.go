// Command quickstart wires authBase v3 with in-memory adapters.
//
// Demo only: replace memoryIdentity/memoryRefreshStore with your DB adapters
// before production use. See docs/refresh-token-store.md for store guarantees.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/WilsonSayago/authBase/v4/core/domain"
	"github.com/WilsonSayago/authBase/v4/core/services"
	"github.com/WilsonSayago/authBase/v4/infra/config/properties"
	"github.com/WilsonSayago/authBase/v4/infra/secundary"
)

func main() {
	cfg := properties.Jwt{
		SecretKey:        strings.Repeat("A", properties.MinSecretBytes),
		RefreshSecret:    strings.Repeat("B", properties.MinSecretBytes),
		ExpirationTime:   1,
		RefreshTokenTime: 24,
		Issuer:           "authbase-example",
		Audience:         "authbase-api",
		LeewaySeconds:    0,
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("jwt config: %v", err)
	}

	identity := newMemoryIdentity()
	validation := secundary.NewValidationService()
	hash, err := validation.HashPassword("correct-horse-battery")
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}
	identity.add(
		domain.NewUserGeneric("user-1", "Ada", "ada@example.com", nil, false, true),
		"ada@example.com",
		hash,
	)

	tokens, err := services.NewTokenManager(cfg)
	if err != nil {
		log.Fatalf("token manager: %v", err)
	}
	authn, err := services.NewAuthenticationService[domain.UserGeneric](
		identity,
		identity,
		newMemoryRefreshStore(),
		validation,
		cfg,
	)
	if err != nil {
		log.Fatalf("authentication service: %v", err)
	}
	authz, err := services.NewAuthorization[domain.UserGeneric, *demoContext](identity, tokens)
	if err != nil {
		log.Fatalf("authorization: %v", err)
	}

	ctx := context.Background()
	access, refresh, err := authn.Login(ctx, "ada@example.com", "correct-horse-battery")
	if err != nil {
		log.Fatalf("login: %v", err)
	}
	fmt.Printf("login ok access_len=%d refresh_len=%d\n", len(access), len(refresh))

	httpCtx := &demoContext{header: "Bearer " + access, req: ctx}
	authz.AuthorizeJWT()(httpCtx)
	if httpCtx.aborted {
		log.Fatalf("authorize aborted: %d %#v", httpCtx.abortCode, httpCtx.abortBody)
	}
	user, ok := authz.GetUserToken(httpCtx)
	if !ok {
		log.Fatal("authorize did not store user")
	}
	fmt.Printf("authorize ok user=%s\n", user.GetId())

	newAccess, newRefresh, err := authn.RefreshToken(ctx, refresh)
	if err != nil {
		log.Fatalf("refresh: %v", err)
	}
	fmt.Printf("refresh ok access_len=%d refresh_len=%d\n", len(newAccess), len(newRefresh))
}
