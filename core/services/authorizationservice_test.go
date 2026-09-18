package services

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/WilsonSayago/authBase/core"
	"github.com/WilsonSayago/authBase/core/domain"
	"github.com/WilsonSayago/authBase/infra/config/properties"
)

type recordingHTTPContext struct {
	header     string
	reqCtx     context.Context
	abortCalls int
	abortCode  int
	abortBody  any
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

func (c *recordingHTTPContext) AbortWithStatusJSON(code int, body interface{}) {
	c.abortCalls++
	c.abortCode = code
	c.abortBody = body
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

func authzTestConfig() properties.Jwt {
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

func newTestAuthorization(t *testing.T, store *fakeIdentityStore) *Authorization[fakeUser, *recordingHTTPContext] {
	t.Helper()
	tokens, err := NewTokenManager(authzTestConfig())
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	authz, err := NewAuthorization[fakeUser, *recordingHTTPContext](store, tokens)
	if err != nil {
		t.Fatalf("NewAuthorization() error = %v", err)
	}
	return authz
}

func assertUnauthorizedAbort(t *testing.T, httpCtx *recordingHTTPContext) {
	t.Helper()
	if httpCtx.abortCalls != 1 {
		t.Fatalf("AbortWithStatusJSON calls = %d, want 1", httpCtx.abortCalls)
	}
	if httpCtx.abortCode != http.StatusUnauthorized {
		t.Fatalf("abort code = %d, want %d", httpCtx.abortCode, http.StatusUnauthorized)
	}
	if httpCtx.nextCalls != 0 {
		t.Fatalf("Next() calls = %d, want 0", httpCtx.nextCalls)
	}
	if _, ok := httpCtx.values[contextUserKey]; ok {
		t.Fatal("user must not be stored on failure")
	}
}

func TestAuthorizeJWTSuccess(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "hash")
	authz := newTestAuthorization(t, store)

	access, _ := mustIssuePair(t, authz.tokens, "user-1")

	httpCtx := newRecordingHTTPContext("Bearer "+access, context.Background())
	authz.AuthorizeJWT()(httpCtx)

	if httpCtx.abortCalls != 0 {
		t.Fatalf("AbortWithStatusJSON calls = %d, want 0", httpCtx.abortCalls)
	}
	if httpCtx.nextCalls != 1 {
		t.Fatalf("Next() calls = %d, want 1", httpCtx.nextCalls)
	}
	got, ok := httpCtx.values[contextUserKey].(fakeUser)
	if !ok || got.id != "user-1" {
		t.Fatalf("stored user = %#v, want user-1", httpCtx.values[contextUserKey])
	}
}

func TestAuthorizeJWTRejectsInactiveUser(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "hash")
	authz := newTestAuthorization(t, store)

	access, _ := mustIssuePair(t, authz.tokens, "user-1")
	store.setActive("user-1", false)

	httpCtx := newRecordingHTTPContext("Bearer "+access, context.Background())
	authz.AuthorizeJWT()(httpCtx)
	assertUnauthorizedAbort(t, httpCtx)
}

func TestAuthorizeJWTPropagatesRequestContext(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "hash")
	authz := newTestAuthorization(t, store)

	access, _ := mustIssuePair(t, authz.tokens, "user-1")

	reqCtx, cancel := context.WithCancel(context.Background())
	cancel()
	httpCtx := newRecordingHTTPContext("Bearer "+access, reqCtx)
	authz.AuthorizeJWT()(httpCtx)

	assertUnauthorizedAbort(t, httpCtx)
	if store.findByIDN != 1 {
		t.Fatalf("FindByID calls = %d, want 1", store.findByIDN)
	}
}

func TestAuthorizeJWTHostileHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
	}{
		{name: "empty", header: ""},
		{name: "whitespace", header: "   "},
		{name: "short", header: "Bear"},
		{name: "basic", header: "Basic abc"},
		{name: "bearer only", header: "Bearer"},
		{name: "bearer spaces only", header: "Bearer    "},
		{name: "extra segments", header: "Bearer token extra"},
		{name: "no scheme", header: "sometoken"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := newFakeIdentityStore()
			authz := newTestAuthorization(t, store)
			httpCtx := newRecordingHTTPContext(tc.header, context.Background())
			authz.AuthorizeJWT()(httpCtx)
			assertUnauthorizedAbort(t, httpCtx)
			if store.findByIDN != 0 {
				t.Fatalf("FindByID must not be called for hostile header %q", tc.name)
			}
		})
	}
}

func TestAuthorizeJWTRejectsInvalidTokens(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "hash")
	authz := newTestAuthorization(t, store)
	access, refresh := mustIssuePair(t, authz.tokens, "user-1")

	t.Run("malformed", func(t *testing.T) {
		t.Parallel()
		local := newTestAuthorization(t, store)
		httpCtx := newRecordingHTTPContext("Bearer not-a-jwt", context.Background())
		local.AuthorizeJWT()(httpCtx)
		assertUnauthorizedAbort(t, httpCtx)
	})

	t.Run("refresh as access", func(t *testing.T) {
		t.Parallel()
		local := newTestAuthorization(t, store)
		httpCtx := newRecordingHTTPContext("Bearer "+refresh, context.Background())
		local.AuthorizeJWT()(httpCtx)
		assertUnauthorizedAbort(t, httpCtx)
	})

	t.Run("foreign signature", func(t *testing.T) {
		t.Parallel()
		foreignCfg := authzTestConfig()
		foreignCfg.SecretKey = strings.Repeat("C", properties.MinSecretBytes)
		foreignCfg.RefreshSecret = strings.Repeat("D", properties.MinSecretBytes)
		foreign, err := NewTokenManager(foreignCfg)
		if err != nil {
			t.Fatalf("NewTokenManager(foreign) error = %v", err)
		}
		foreignAccess, _ := mustIssuePair(t, foreign, "user-1")
		local := newTestAuthorization(t, store)
		httpCtx := newRecordingHTTPContext("Bearer "+foreignAccess, context.Background())
		local.AuthorizeJWT()(httpCtx)
		assertUnauthorizedAbort(t, httpCtx)
	})

	t.Run("expired", func(t *testing.T) {
		t.Parallel()
		fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		expiredTM, err := newTokenManagerForTest(authzTestConfig(), func() time.Time { return fixed }, nil)
		if err != nil {
			t.Fatalf("newTokenManagerForTest() error = %v", err)
		}
		expiredAccess, _ := mustIssuePair(t, expiredTM, "user-1")
		expiredTM.now = func() time.Time { return fixed.Add(2 * time.Hour) }

		localStore := newFakeIdentityStore()
		localStore.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "hash")
		local := &Authorization[fakeUser, *recordingHTTPContext]{users: localStore, tokens: expiredTM}
		httpCtx := newRecordingHTTPContext("Bearer "+expiredAccess, context.Background())
		local.AuthorizeJWT()(httpCtx)
		assertUnauthorizedAbort(t, httpCtx)
	})

	t.Run("missing user", func(t *testing.T) {
		t.Parallel()
		local := newTestAuthorization(t, newFakeIdentityStore())
		httpCtx := newRecordingHTTPContext("Bearer "+access, context.Background())
		local.AuthorizeJWT()(httpCtx)
		assertUnauthorizedAbort(t, httpCtx)
	})

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()
		localStore := newFakeIdentityStore()
		localStore.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "hash")
		localStore.errByID = core.ErrUnavailable
		local := newTestAuthorization(t, localStore)
		httpCtx := newRecordingHTTPContext("Bearer "+access, context.Background())
		local.AuthorizeJWT()(httpCtx)
		assertUnauthorizedAbort(t, httpCtx)
	})
}

func TestAuthorizeJWTAcceptsCaseInsensitiveBearer(t *testing.T) {
	t.Parallel()

	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "hash")
	authz := newTestAuthorization(t, store)
	access, _ := mustIssuePair(t, authz.tokens, "user-1")

	httpCtx := newRecordingHTTPContext("bearer "+access, context.Background())
	authz.AuthorizeJWT()(httpCtx)
	if httpCtx.nextCalls != 1 || httpCtx.abortCalls != 0 {
		t.Fatalf("expected success for lowercase bearer, abort=%d next=%d", httpCtx.abortCalls, httpCtx.nextCalls)
	}
}

func TestGetUserTokenSafe(t *testing.T) {
	t.Parallel()

	authz := &Authorization[fakeUser, *recordingHTTPContext]{}
	httpCtx := newRecordingHTTPContext("", context.Background())

	if _, ok := authz.GetUserToken(httpCtx); ok {
		t.Fatal("missing key should return false")
	}

	httpCtx.values[contextUserKey] = nil
	if _, ok := authz.GetUserToken(httpCtx); ok {
		t.Fatal("nil value should return false")
	}

	httpCtx.values[contextUserKey] = "not-a-user"
	if _, ok := authz.GetUserToken(httpCtx); ok {
		t.Fatal("wrong type should return false")
	}

	want := fakeUser{id: "user-1", active: true}
	httpCtx.values[contextUserKey] = want
	got, ok := authz.GetUserToken(httpCtx)
	if !ok || got.id != want.id {
		t.Fatalf("GetUserToken() = (%#v, %v), want user-1", got, ok)
	}
}

func TestPoliciesGuardMatrix(t *testing.T) {
	t.Parallel()

	authz := &Authorization[fakeUser, *recordingHTTPContext]{}

	tests := []struct {
		name       string
		user       any
		hasUser    bool
		fnValidate func(fakeUser, string, domain.OperationEnum) bool
		wantAbort  int
		wantNext   int
		wantCode   int
	}{
		{
			name:      "missing user",
			hasUser:   false,
			wantAbort: 1,
			wantCode:  http.StatusUnauthorized,
		},
		{
			name:      "inactive user",
			hasUser:   true,
			user:      fakeUser{id: "u1", active: false, isAdmin: true},
			wantAbort: 1,
			wantCode:  http.StatusUnauthorized,
		},
		{
			name:      "active without permission",
			hasUser:   true,
			user:      fakeUser{id: "u1", active: true},
			wantAbort: 1,
			wantCode:  http.StatusForbidden,
		},
		{
			name:     "active admin",
			hasUser:  true,
			user:     fakeUser{id: "u1", active: true, isAdmin: true},
			wantNext: 1,
		},
		{
			name:    "active with permission",
			hasUser: true,
			user: fakeUser{
				id:     "u1",
				active: true,
				hasPermission: func(entity string, operation domain.OperationEnum) bool {
					return entity == "users" && operation == domain.READ
				},
			},
			wantNext: 1,
		},
		{
			name:    "custom callback denies",
			hasUser: true,
			user:    fakeUser{id: "u1", active: true, isAdmin: true},
			fnValidate: func(fakeUser, string, domain.OperationEnum) bool {
				return false
			},
			wantAbort: 1,
			wantCode:  http.StatusForbidden,
		},
		{
			name:    "custom callback allows active user",
			hasUser: true,
			user:    fakeUser{id: "u1", active: true},
			fnValidate: func(fakeUser, string, domain.OperationEnum) bool {
				return true
			},
			wantNext: 1,
		},
		{
			name:    "custom callback cannot skip inactive",
			hasUser: true,
			user:    fakeUser{id: "u1", active: false},
			fnValidate: func(fakeUser, string, domain.OperationEnum) bool {
				return true
			},
			wantAbort: 1,
			wantCode:  http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			httpCtx := newRecordingHTTPContext("", context.Background())
			if tc.hasUser {
				httpCtx.values[contextUserKey] = tc.user
			}
			called := 0
			handler := authz.PoliciesGuard(func(*recordingHTTPContext) {
				called++
			}, tc.fnValidate, "users", domain.READ)
			handler(httpCtx)

			if httpCtx.abortCalls != tc.wantAbort {
				t.Fatalf("abort calls = %d, want %d", httpCtx.abortCalls, tc.wantAbort)
			}
			if tc.wantAbort > 0 && httpCtx.abortCode != tc.wantCode {
				t.Fatalf("abort code = %d, want %d", httpCtx.abortCode, tc.wantCode)
			}
			if called != tc.wantNext {
				t.Fatalf("handler calls = %d, want %d", called, tc.wantNext)
			}
		})
	}
}

func FuzzAuthorizeHeader(f *testing.F) {
	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "hash")
	tokens, err := NewTokenManager(authzTestConfig())
	if err != nil {
		f.Fatal(err)
	}
	authz, err := NewAuthorization[fakeUser, *recordingHTTPContext](store, tokens)
	if err != nil {
		f.Fatal(err)
	}

	access, _ := mustIssuePair(f, authz.tokens, "user-1")
	f.Add("")
	f.Add("Bearer")
	f.Add("Bearer ")
	f.Add("Basic abc")
	f.Add("Bearer " + access)
	f.Add("bearer " + access)
	f.Add("Bearer token extra")
	f.Add(string([]byte{0x00, 0x01, 0xff}))

	f.Fuzz(func(t *testing.T, header string) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatalf("AuthorizeJWT panicked: %v", rec)
			}
		}()
		httpCtx := newRecordingHTTPContext(header, context.Background())
		authz.AuthorizeJWT()(httpCtx)
		if httpCtx.abortCalls+httpCtx.nextCalls != 1 {
			t.Fatalf("expected exactly one terminal action, abort=%d next=%d", httpCtx.abortCalls, httpCtx.nextCalls)
		}
	})
}
