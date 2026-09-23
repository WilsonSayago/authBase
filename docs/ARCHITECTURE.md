# authBase — Architecture

`authBase` is a **hexagonal authentication and authorization library** for Go
microservices. It is not an HTTP server, database, or full identity platform:
applications provide ports and wire constructors explicitly.

## What belongs in the library

| Concern | Package API | Notes |
|---------|-------------|--------|
| Domain models | `core/domain` | `IUserGeneric`, roles, permissions, `RefreshSession`, `HashRefreshToken` |
| Use-case contracts | `core` | `AuthenticationUseCase`, `AuthorizationUseCase`, `Context`, sentinel errors |
| Ports | `core/port` | `UserReader`, `CredentialReader`, `RefreshTokenStore`, `ValidationPort`, `RolePort` |
| Authentication | `services.NewAuthenticationService` | Login, refresh rotation, validate, revoke family |
| Authorization | `services.NewAuthorization` | Panic-safe `AuthorizeJWT` + `PoliciesGuard` |
| JWT | `services.NewTokenManager`, `properties.Jwt` | Typed claims; refresh minting unexported |
| Password hashing | `infra/secundary.ValidationService` | bcrypt adapter for `ValidationPort` |

## What belongs in each microservice

| Concern | Location | Notes |
|---------|----------|--------|
| Composition root | `internal/bootstrap/` | Construct services with app-owned adapters |
| Identity / credentials adapters | `internal/infra/...` | Implement `UserReader` + `CredentialReader` |
| Refresh session store | `internal/infra/...` | Implement `RefreshTokenStore` with atomic `Rotate` |
| HTTP adapter for `core.Context` | framework glue | Gin/Echo/chi wrapper; not shipped by authBase |
| JWT config load | env / YAML → `properties.Jwt` | Call `Validate()` at process start |
| Thin entrypoint | `cmd/main.go` | Wire and serve |

## Typical wiring flow

```text
cmd / bootstrap
    ├── cfg := properties.Jwt{…}; cfg.Validate()
    ├── tokens, _ := services.NewTokenManager(cfg)
    ├── authn, _ := services.NewAuthenticationService(users, creds, refresh, validation, cfg)
    ├── authz, _ := services.NewAuthorization(users, tokens)
    ├── POST /login    → authn.Login
    ├── POST /refresh  → authn.RefreshToken
    └── protected routes → authz.AuthorizeJWT() → PoliciesGuard(…)
```

See the compilable example at [`examples/quickstart`](../examples/quickstart).

## Dependency direction

```text
  cmd / bootstrap  →  authBase/v3 (services, properties)
         ↓
    app adapters implement core/port
         ↓
    app domain / infra (DB, HTTP framework)
```

Application **domain** packages should not import authBase. Only bootstrap,
HTTP adapters, and infrastructure wiring may depend on it.

`AuthenticationService` owns the issuance boundary: it persists a refresh
session (`Create` / `Rotate`) before returning tokens. `TokenManager` verifies
access/refresh JWTs; pair helpers are unexported so callers cannot mint
unregister refresh tokens.

## Diagrams

Interactive Archify diagrams (open the HTML in a browser):

| View | HTML |
|------|------|
| Component map (library vs consumer) | [diagrams/authbase-architecture.html](diagrams/authbase-architecture.html) |
| Login + refresh workflow | [diagrams/authbase-login-refresh.html](diagrams/authbase-login-refresh.html) |
| Refresh session lifecycle | [diagrams/authbase-refresh-lifecycle.html](diagrams/authbase-refresh-lifecycle.html) |

Index and regenerate notes: [diagrams/README.md](diagrams/README.md).

## Versioning

- **v3.x** — module path `github.com/WilsonSayago/authBase/v3` (required by Go for
  major ≥ 2). Pre-v3 tokens and singleton constructors are not supported.
- Migration: [MIGRATION_V3.md](MIGRATION_V3.md).
- Publish gates: [RELEASE_CHECKLIST.md](RELEASE_CHECKLIST.md).

## Related docs

- [README.md](../README.md) — usage, threat model, HTTP status mapping
- [diagrams/README.md](diagrams/README.md) — interactive Archify diagrams
- [refresh-token-store.md](refresh-token-store.md) — store contract
- [SECURITY.md](SECURITY.md) — reporting and logging rules
- [MIGRATION_V3.md](MIGRATION_V3.md) — API old → new
- [examples/quickstart](../examples/quickstart) — in-memory wiring demo
