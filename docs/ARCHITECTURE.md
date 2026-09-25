# authBase — Architecture

`authBase` is a **hexagonal authentication and authorization library** for Go
microservices. It is not an HTTP server, database, or full identity platform:
applications provide ports and wire constructors explicitly.

## What belongs in the library

| Concern | Package API | Notes |
|---------|-------------|--------|
| Domain models | `core/domain` | `IUserGeneric`, roles, permissions, `RefreshSession`, `HashRefreshToken` |
| Pagination contracts | `core/domain` | `CursorKey`, `PageRequest`, `Page[T]` for forward-only keyset pagination |
| Use-case contracts | `core` | `AuthenticationUseCase`, additive `SessionIssuer` / `SessionManager`, `AuthorizationUseCase`, `Context`, sentinel errors |
| Ports | `core/port` | `UserReader`, `CredentialReader`, `RefreshTokenStore`, optional session lifecycle capabilities, `ValidationPort`, `RolePort` |
| Authentication | `services.NewAuthenticationService` | Login, privileged session establishment, refresh rotation, validate, revoke |
| Authorization | `services.NewAuthorization` | Panic-safe `AuthorizeJWT` + `PoliciesGuard` |
| JWT | `services.NewTokenManager`, `properties.Jwt` | Typed claims; refresh minting unexported |
| Password hashing | `infra/secundary.ValidationService` | Argon2id PHC for new hashes; bcrypt legacy verification |
| Role operations | `services.RoleService` | Context-aware CRUD, `SetActive`, paginated `GetRoles` |

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
    ├── roles := services.GetRoleServiceInstance(rolePort)
    ├── POST /login    → authn.Login
    ├── POST /refresh  → authn.RefreshToken
    ├── trusted auth flow → authn.EstablishSession
    ├── GET /roles      → roles.GetRoles(ctx, PageRequest)
    └── protected routes → authz.AuthorizeJWT() → PoliciesGuard(…)
```

See the compilable example at [`examples/quickstart`](../examples/quickstart).

## Dependency direction

```text
  cmd / bootstrap  →  authBase/v4 (services, properties)
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

User-wide revocation and expiry cleanup are optional adapter capabilities kept
separate from `RefreshTokenStore`; v4 stores remain source compatible while
applications can opt into `RevokeUserSessions` and retention-aware cleanup.

## Diagrams

Interactive Archify diagrams (open the HTML in a browser):

| View | HTML |
|------|------|
| Component map (library vs consumer) | [diagrams/authbase-architecture.html](diagrams/authbase-architecture.html) |
| Login + refresh workflow | [diagrams/authbase-login-refresh.html](diagrams/authbase-login-refresh.html) |
| Refresh session lifecycle | [diagrams/authbase-refresh-lifecycle.html](diagrams/authbase-refresh-lifecycle.html) |

Index and regenerate notes: [diagrams/README.md](diagrams/README.md).

## Versioning

- **v4.x** — module path `github.com/WilsonSayago/authBase/v4` (required by Go for
  major ≥ 2). Role listings use typed keyset pagination.
- v3 → v4: [MIGRATION_V4.md](MIGRATION_V4.md).
- Pre-v3 migration: [MIGRATION_V3.md](MIGRATION_V3.md).
- Publish gates: [RELEASE_CHECKLIST.md](RELEASE_CHECKLIST.md).

## Related docs

- [README.md](../README.md) — usage, threat model, HTTP status mapping
- [diagrams/README.md](diagrams/README.md) — interactive Archify diagrams
- [refresh-token-store.md](refresh-token-store.md) — store contract
- [SECURITY.md](SECURITY.md) — reporting and logging rules
- [MIGRATION_V4.md](MIGRATION_V4.md) — v3 → v4 pagination changes
- [MIGRATION_V3.md](MIGRATION_V3.md) — API old → new
- [examples/quickstart](../examples/quickstart) — in-memory wiring demo
