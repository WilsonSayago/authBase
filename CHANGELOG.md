# Changelog

All notable changes to authBase are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project intends to follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html) starting with v3.

## [Unreleased]

### Added

- Additive `SessionIssuer`, `SessionManager`, `UserSessionRevoker`, and
  `ExpiredSessionPurger` contracts for user-wide session lifecycle management.
- `AuthenticationService.EstablishSession` for already-authenticated active
  identities and `RevokeUserSessions` for optional store-backed revocation.

### Changed

- Login and privileged session establishment now share one persist-before-return
  issuance path without changing the v4 baseline interfaces.

### Fixed

- Complete the temporary consumer's transitive checksums before compiling the
  published-module smoke test.

## [4.0.1] - 2026-09-25

### Build

- Added a reproducible `make verify` gate with pinned analysis, vulnerability,
  module-integrity, quickstart, race, and secret-scanning checks.
- Added weekly Go module and GitHub Actions dependency updates.
- Added a public-proxy consumer smoke test that rejects workspace replacements.

## [4.0.0] - 2026-09-24

### Added

- Generic keyset pagination types: `CursorKey`, `PageRequest`, and `Page[T]`.

### Changed

- Module path is now `github.com/WilsonSayago/authBase/v4`.
- `RolePort.FindAll` and `RoleUseCase.GetRoles` now use keyset pagination
  instead of page size, offset, and exact totals.

## [3.0.1] - 2026-09-24

Patch release so Go module consumers can pin the post-`v3.0.0` API. The
GitHub `v3.0.0` tag was moved after the module proxy recorded the original
tree; **do not retag `v3.0.0`**. Depend on `v3.0.1` (or newer) for the
surface below.

### Changed

- `RolePort` / `RoleUseCase` take `context.Context` on every method.
- Replaced toggle-style `ChangeStatus` with explicit `SetActive(ctx, id, active)`.

### Documentation

- Architecture guide and Archify diagrams under `docs/`.
- Removed completed implementation `plans/` backlog.

## [3.0.0] - 2026-09-18

First published `/v3` module (immutable in the Go module proxy).

### Added

- Module path `github.com/WilsonSayago/authBase/v3`.
- `TokenManager` with typed claims, issuer/audience, and dual secrets.
- `RefreshTokenStore` port with atomic rotation and family revoke.
- Active-identity enforcement on login, validate, authorize, and refresh.
- Credential isolation via `CredentialReader` / `CredentialRecord`.
- Panic-safe authorization middleware with 401/403 separation.
- CI matrix for Go 1.26.8 / 1.27.1 including race, vet, and govulncheck.
- Quickstart example under `examples/quickstart`.
- Migration, security, and refresh-store documentation.

### Changed

- Constructors return errors and no longer rely on initModules singletons.
- Refresh tokens are issued only through `AuthenticationService` after store
  persistence.

### Security

- JWT baseline `github.com/golang-jwt/jwt/v5 v5.3.1`.
- `golang.org/x/crypto v0.57.0`.
- Toolchain directive `go1.27.1` with minimum language `go 1.26.0`.

[Unreleased]: https://github.com/WilsonSayago/authBase/compare/v4.0.1...HEAD
[4.0.1]: https://github.com/WilsonSayago/authBase/compare/v4.0.0...v4.0.1
[4.0.0]: https://github.com/WilsonSayago/authBase/releases/tag/v4.0.0
[3.0.1]: https://github.com/WilsonSayago/authBase/releases/tag/v3.0.1
[3.0.0]: https://github.com/WilsonSayago/authBase/releases/tag/v3.0.0
