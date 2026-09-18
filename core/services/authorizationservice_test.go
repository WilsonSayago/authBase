package services

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/WilsonSayago/authBase/infra/config/properties"
)

type recordingHTTPContext struct {
	header     string
	reqCtx     context.Context
	abortCalls int
	abortCode  int
	nextCalls  int
	values     map[string]any
}

func newRecordingHTTPContext(header string, reqCtx context.Context) *recordingHTTPContext {
	return &recordingHTTPContext{
		header: header,
		reqCtx: reqCtx,
		values: make(map[string]any),
	}
}

func (c *recordingHTTPContext) GetHeader(key string) string {
	if strings.EqualFold(key, "Authorization") {
		return c.header
	}
	return ""
}

func (c *recordingHTTPContext) Set(key string, value interface{}) {
	c.values[key] = value
}

func (c *recordingHTTPContext) AbortWithStatusJSON(code int, _ interface{}) {
	c.abortCalls++
	c.abortCode = code
}

func (c *recordingHTTPContext) Next() {
	c.nextCalls++
}

func (c *recordingHTTPContext) Get(key string) (any, bool) {
	value, ok := c.values[key]
	return value, ok
}

func (c *recordingHTTPContext) Status(int) {}

func (c *recordingHTTPContext) RequestContext() context.Context {
	if c.reqCtx != nil {
		return c.reqCtx
	}
	return context.Background()
}

func TestAuthorizeJWTRejectsInactiveUser(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")

	cfg := properties.Jwt{
		SecretKey:        strings.Repeat("A", properties.MinSecretBytes),
		RefreshSecret:    strings.Repeat("B", properties.MinSecretBytes),
		ExpirationTime:   1,
		RefreshTokenTime: 24,
		Issuer:           "authbase-test",
		Audience:         "authbase-clients",
		LeewaySeconds:    0,
	}
	authz, err := NewAuthorization[fakeUser, *recordingHTTPContext](store, cfg)
	if err != nil {
		t.Fatalf("NewAuthorization() error = %v", err)
	}

	access, _, err := authz.tokens.IssuePair("user-1")
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}

	store.setActive("user-1", false)

	httpCtx := newRecordingHTTPContext("Bearer "+access, context.Background())
	authz.AuthorizeJWT()(httpCtx)

	if httpCtx.abortCalls != 1 {
		t.Fatalf("AbortWithStatusJSON calls = %d, want 1", httpCtx.abortCalls)
	}
	if httpCtx.abortCode != http.StatusUnauthorized {
		t.Fatalf("abort code = %d, want %d", httpCtx.abortCode, http.StatusUnauthorized)
	}
	if httpCtx.nextCalls != 0 {
		t.Fatalf("Next() calls = %d, want 0", httpCtx.nextCalls)
	}
	if _, ok := httpCtx.values["user"]; ok {
		t.Fatal("inactive user must not be stored in context")
	}
}

func TestAuthorizeJWTPropagatesRequestContext(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")

	cfg := properties.Jwt{
		SecretKey:        strings.Repeat("A", properties.MinSecretBytes),
		RefreshSecret:    strings.Repeat("B", properties.MinSecretBytes),
		ExpirationTime:   1,
		RefreshTokenTime: 24,
		Issuer:           "authbase-test",
		Audience:         "authbase-clients",
		LeewaySeconds:    0,
	}
	authz, err := NewAuthorization[fakeUser, *recordingHTTPContext](store, cfg)
	if err != nil {
		t.Fatalf("NewAuthorization() error = %v", err)
	}

	access, _, err := authz.tokens.IssuePair("user-1")
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}

	reqCtx, cancel := context.WithCancel(context.Background())
	cancel()
	httpCtx := newRecordingHTTPContext("Bearer "+access, reqCtx)
	authz.AuthorizeJWT()(httpCtx)

	if httpCtx.abortCalls != 1 {
		t.Fatalf("AbortWithStatusJSON calls = %d, want 1", httpCtx.abortCalls)
	}
	if httpCtx.nextCalls != 0 {
		t.Fatalf("Next() calls = %d, want 0", httpCtx.nextCalls)
	}
	if store.findByIDN != 1 {
		t.Fatalf("FindByID calls = %d, want 1", store.findByIDN)
	}
}
