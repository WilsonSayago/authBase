# Security policy

## Supported versions

Only the upcoming **v3** line is intended for new integrations. Historical
`v1`/`v2` tags are not a supported security baseline for this module path.

Support details will be restated when a release tag is published.

## Reporting a vulnerability

**Do not** open a public GitHub issue for security reports.

> **BLOCKER for release:** replace this marker with a maintainer-confirmed
> private channel (for example a security email or GitHub Security Advisories
> on the canonical repository) before declaring authBase release-ready.

Until that channel exists, treat unreleased builds as internal-only.

## What never to log

Never write these to logs, traces, metrics labels, or error messages returned to
clients:

- access or refresh JWT strings
- password plaintext or password hashes
- JWT signing secrets (`secret-key`, `refresh-secret`)
- raw refresh session material (store only SHA-256 hashes)

Safe to log at appropriate levels: user id (subject), refresh `family_id`,
token `jti`, error sentinel names, HTTP status.

## Hardening expectations for adapters

- Persist refresh sessions only as hashes (`domain.HashRefreshToken`).
- Make `Rotate` atomic; on `ErrRefreshConsumed`, revoke the family.
- Keep access tokens short-lived; treat refresh as the session boundary.
- Validate JWT config via `properties.Jwt.Validate()` at process start.
