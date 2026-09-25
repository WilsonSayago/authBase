# Security policy

## Supported versions

Only the **v4** line is intended for new integrations. Historical `v1`/`v2`
tags and the superseded `v3` API are not a supported security baseline for new
deployments.

Security fixes target the current major line. Once v4 tags are published,
consumers should track the latest `v4.x` release rather than `main`.

## Reporting a vulnerability

**Do not** open a public GitHub issue for security reports.

Report privately with **GitHub Security Advisories** on the canonical repository:

- Open a draft advisory:
  https://github.com/WilsonSayago/authBase/security/advisories/new
- Or use **Report a vulnerability** on the repo Security tab (Private
  vulnerability reporting).

Include impact, affected versions/commits, and a minimal reproduction when
possible. Allow reasonable time for a fix before any public disclosure.

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
