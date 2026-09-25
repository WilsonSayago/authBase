# authBase

Hexagonal Go authentication and authorization library: login, JWT access/refresh
tokens, active-identity checks, and panic-safe HTTP middleware adapters.

authBase does **not** ship HTTP servers, databases, or a production refresh-token
store. Applications provide ports (`UserReader`, `CredentialReader`,
`RefreshTokenStore`, `ValidationPort`, `RolePort`) and wire constructors
explicitly.

## Go versions

| | Version |
|---|---|
| Minimum language (`go` directive) | 1.26.0 |
| Recommended toolchain | go1.27.1 |
| CI matrix | 1.26.8, 1.27.1 |

## Install

Install the published v4 release:

```sh
go get github.com/WilsonSayago/authBase/v4@v4.0.1
```

Pin an explicit `v4.x.y` tag instead of tracking `main`.

```go
import (
    "github.com/WilsonSayago/authBase/v4/core/services"
    "github.com/WilsonSayago/authBase/v4/infra/config/properties"
)
```

## Quickstart

See the compilable example at [`examples/quickstart`](examples/quickstart):

```sh
env GOWORK=off go run ./examples/quickstart
env GOWORK=off go test ./examples/quickstart -count=1
```

It shows validated JWT config, in-memory ports, `NewAuthenticationService`,
`NewAuthorization` + `AuthorizeJWT`, login, and atomic refresh rotation.

Production adapters must implement `port.RefreshTokenStore` with the guarantees
in [`docs/refresh-token-store.md`](docs/refresh-token-store.md).

## Cursor pagination (v4)

`core/domain` exports `CursorKey`, `PageRequest`, and `Page[T]` for forward-only
keyset pagination. `RolePort.FindAll` and `RoleUseCase.GetRoles` accept
`PageRequest` and return `Page[domain.Role]`; adapters own cursor encoding and
the stable datastore ordering.

See [`docs/MIGRATION_V4.md`](docs/MIGRATION_V4.md) for the v3 → v4 API change.

## Session lifecycle capabilities

`AuthenticationService.EstablishSession` is the privileged issuance boundary
for callers that already authenticated a user. It verifies the active identity
and persists the refresh session before returning tokens.

Refresh stores may implement the additive `port.UserSessionRevoker` and
`port.ExpiredSessionPurger` capabilities. Existing v4 implementations of
`port.RefreshTokenStore` remain source compatible.

## Threat model (summary)

- **Bearer access tokens**: short-lived; middleware returns 401 on missing,
  malformed, expired, inactive, or unknown subjects.
- **Refresh tokens**: must be persisted by hash before return; rotation is
  atomic; replay of a consumed token revokes the family.
- **Secrets**: access and refresh signing keys must differ and be at least 32
  bytes; never log tokens, password hashes, or raw secrets.
- **Credentials**: password material lives only in `CredentialRecord`, not on
  `IUserGeneric`.
- **Revocation**: call `RevokeRefreshFamily` for one family or
  `RevokeUserSessions` for account-wide compromise; access tokens remain valid
  until expiry unless you add an extra denylist outside authBase.

## Errors and HTTP status

| Situation | Library signal | Typical HTTP |
|---|---|---|
| Bad login (unknown user, wrong password, inactive) | `core.ErrInvalidCredentials` | 401 |
| Bad/replayed refresh | `core.ErrInvalidRefresh` | 401 |
| Missing/invalid Bearer or inactive subject in JWT middleware | abort body `{"error":"unauthorized"}` | 401 |
| Authenticated but lacking permission | abort body `{"error":"forbidden"}` | 403 |
| Inactive identity during `ValidateToken` | `core.ErrInactiveIdentity` | 401 |

Do not distinguish failure reasons to clients for login/refresh.

## Compatibility

v4 is a **breaking API** release because role listings moved from offset
pagination to typed keyset pagination. JWT claims and refresh-session behavior
are unchanged from v3, so upgrading v3 → v4 does not itself force a re-login
when signing configuration remains the same.

- v3 → v4: [`docs/MIGRATION_V4.md`](docs/MIGRATION_V4.md)
- pre-v3 → modern API: [`docs/MIGRATION_V3.md`](docs/MIGRATION_V3.md)

## Verify

```sh
GOWORK=off make verify
```

After a release is visible through the public Go proxy, verify it from an
isolated temporary consumer with no workspace or `replace` directive:

```sh
GOWORK=off make consumer-published AUTHBASE_VERSION=v4.0.0
```

## Docs

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — library vs microservice responsibilities
- [`docs/diagrams/`](docs/diagrams/) — interactive architecture / login-refresh / lifecycle diagrams (Archify)
- [`docs/MIGRATION_V4.md`](docs/MIGRATION_V4.md) — v3 → v4 keyset pagination
- [`docs/MIGRATION_V3.md`](docs/MIGRATION_V3.md) — API old → new
- [`docs/SECURITY.md`](docs/SECURITY.md) — reporting and logging rules
- [`docs/refresh-token-store.md`](docs/refresh-token-store.md) — store contract
- [`docs/RELEASE_CHECKLIST.md`](docs/RELEASE_CHECKLIST.md) — publish gates
- [`CHANGELOG.md`](CHANGELOG.md)

## License

MIT — see [`LICENSE`](LICENSE).

## Release status

`v4.0.1` is published on the canonical remote and resolves through the public
Go module proxy. The external-consumer smoke test verifies the cached module
without a workspace or `replace`; see
[`docs/RELEASE_CHECKLIST.md`](docs/RELEASE_CHECKLIST.md).
