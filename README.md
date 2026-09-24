# authBase

Hexagonal Go authentication and authorization library: login, JWT access/refresh
tokens, active-identity checks, and panic-safe HTTP middleware adapters.

authBase does **not** ship HTTP servers, databases, or a production refresh-token
store. Applications provide ports (`UserReader`, `CredentialReader`,
`RefreshTokenStore`, `ValidationPort`) and wire constructors explicitly.

## Go versions

| | Version |
|---|---|
| Minimum language (`go` directive) | 1.26.0 |
| Recommended toolchain | go1.27.1 |
| CI matrix | 1.26.8, 1.27.1 |

## Install (after a v3 tag exists)

```sh
go get github.com/WilsonSayago/authBase/v4@v4.0.0
```

Until a release tag is published, depend on a commit or a local `replace`. Do
not treat this README as confirmation that `v3.0.0` already exists on the
module proxy.

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

## Threat model (summary)

- **Bearer access tokens**: short-lived; middleware returns 401 on missing,
  malformed, expired, inactive, or unknown subjects.
- **Refresh tokens**: must be persisted by hash before return; rotation is
  atomic; replay of a consumed token revokes the family.
- **Secrets**: access and refresh signing keys must differ and be at least 32
  bytes; never log tokens, password hashes, or raw secrets.
- **Credentials**: password material lives only in `CredentialRecord`, not on
  `IUserGeneric`.
- **Revocation**: call `RevokeRefreshFamily` on logout/compromise; access tokens
  remain valid until expiry unless you add an extra denylist outside authBase.

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

v3 is a **breaking** major. Pre-v3 tokens, singleton constructors, and module
path `github.com/WilsonSayago/authBase` (without `/v4`) are not supported.
Migration steps: [`docs/MIGRATION_V3.md`](docs/MIGRATION_V3.md).

## Verify

```sh
env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test ./...
env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go test -race ./...
env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off go vet ./...
env GOCACHE=/private/tmp/authbase-go-cache GOWORK=off GOTOOLCHAIN=go1.27.1 \
  go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
```

## Docs

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — library vs microservice responsibilities
- [`docs/diagrams/`](docs/diagrams/) — interactive architecture / login-refresh / lifecycle diagrams (Archify)
- [`docs/MIGRATION_V3.md`](docs/MIGRATION_V3.md) — API old → new
- [`docs/SECURITY.md`](docs/SECURITY.md) — reporting and logging rules
- [`docs/refresh-token-store.md`](docs/refresh-token-store.md) — store contract
- [`docs/RELEASE_CHECKLIST.md`](docs/RELEASE_CHECKLIST.md) — publish gates
- [`CHANGELOG.md`](CHANGELOG.md)

## License

MIT — see [`LICENSE`](LICENSE).

## Release status

Module path `/v4`, MIT license, and private reporting via GitHub Security
Advisories are in place. Publishing the `v3.0.0` tag remains a separate manual
step — follow [`docs/RELEASE_CHECKLIST.md`](docs/RELEASE_CHECKLIST.md).
