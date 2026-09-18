# Changelog

All notable changes to authBase are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project intends to follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html) starting with v3.

## [Unreleased]

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

## [3.0.0] - Unreleased

Prepared breaking major for a valid Go module path. **Not published yet** —
no date, no tag. Publish only after [`docs/RELEASE_CHECKLIST.md`](docs/RELEASE_CHECKLIST.md)
is fully satisfied (license, security contact, canonical remote, CI green).

[Unreleased]: https://github.com/WilsonSayago/authBase/compare/HEAD...HEAD
[3.0.0]: https://github.com/WilsonSayago/authBase/releases/tag/v3.0.0
