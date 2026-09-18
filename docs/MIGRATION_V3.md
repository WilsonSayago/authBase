# Migrating to authBase v3

This guide maps pre-v3 / mid-series APIs to the frozen v3 surface. Tokens issued
before v3 are **not** compatible: claims, secrets policy, and refresh sessions
changed. Plan a forced re-login.

## Module path

| Before | After |
|---|---|
| `github.com/WilsonSayago/authBase` | `github.com/WilsonSayago/authBase/v3` |
| (invalid) tags `v2.0.x` without `/v2` path | Do not reuse; publish `v3.0.0` with `/v3` |

Update every import, including transitive application packages.

```go
import (
    "github.com/WilsonSayago/authBase/v3/core"
    "github.com/WilsonSayago/authBase/v3/core/services"
)
```

## Singletons and initModules

| Before | After |
|---|---|
| Global singleton services via initModules | Removed from authBase |
| Shared mutable instances across apps | Construct per process with DI |

Prefer:

```go
svc, err := services.NewAuthenticationService[...](...)
authz, err := services.NewAuthorization[...](...)
tm, err := services.NewTokenManager(cfg)
```

`GetAuthenticationInstance` remains only as a thin deprecated wrapper that
delegates to `NewAuthenticationService` and still returns an error.

## Constructors return errors

Constructors validate dependencies and JWT config. They never panic on bad
config and never return process-wide singletons.

## JWT / TokenManager

| Before | After |
|---|---|
| Ad-hoc signing/parsing in services | `TokenManager` owns verify; auth service owns persist+issue |
| Shared secret patterns | Distinct access/refresh secrets, issuer, audience, leeway |
| Untyped map claims | `services.TokenClaims` with `token_type` and refresh `family_id` |
| Public `IssuePair` helpers | Unexported; only `AuthenticationService` returns refresh tokens after store writes |

Access and refresh tokens from older majors will fail `ParseAccess` /
`ParseRefresh`.

## Credentials and active users

| Before | After |
|---|---|
| Password fields on user models | `port.CredentialRecord` + `CredentialReader` |
| Inactive users sometimes accepted | Login / ValidateToken / AuthorizeJWT / Refresh require active |

Public login failures collapse to `core.ErrInvalidCredentials`.

## context.Context

Authentication and identity I/O ports take `context.Context`:

- `AuthenticationUseCase.Login|RefreshToken|ValidateToken|RevokeRefreshFamily`
- `UserReader.FindByID`
- `CredentialReader.FindCredentialsByUsername`
- `RefreshTokenStore.Create|Rotate|RevokeFamily`

`core.Context.RequestContext()` supplies the request-scoped context to
authorization middleware.

## Refresh rotation

| Before | After |
|---|---|
| Stateless refresh re-issue | `RefreshTokenStore` with atomic `Rotate` |
| Reusable stolen refresh until expiry | Replay → `ErrRefreshConsumed` → family revoke → `ErrInvalidRefresh` |

Implement the contract in [`refresh-token-store.md`](refresh-token-store.md).
In-memory maps are fine for tests/examples only.

## Middleware and HTTP

| Before | After |
|---|---|
| Panics / unclear status | `AuthorizeJWT` / `PoliciesGuard` recover-safe paths returning JSON |
| Mixed authz failures | **401** unauthorized (no/invalid token, inactive) vs **403** forbidden (permission) |

Use `GetUserToken`; do not read internal context keys.

## Errors to handle

- `ErrInvalidCredentials`, `ErrInvalidRefresh`, `ErrInactiveIdentity`
- `ErrNotFound`, `ErrUnavailable` (infrastructure mapping)
- Store: `ErrRefreshNotFound`, `ErrRefreshConsumed`, `ErrRefreshFamilyRevoked`
  (mapped to `ErrInvalidRefresh` at the public refresh boundary)

## Checklist for consumers

1. Change module path to `/v3` and run `go mod tidy`.
2. Replace singletons with `New*` constructors; handle errors.
3. Split credential storage from user profile ports.
4. Implement atomic `RefreshTokenStore`.
5. Force re-authentication (invalidate old JWTs).
6. Align HTTP mapping with 401/403 table in the README.
7. Confirm Go ≥ 1.26.0 and dual secrets ≥ 32 bytes.
