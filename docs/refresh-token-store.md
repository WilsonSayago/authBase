# Refresh token store contract

Adapters that implement `port.RefreshTokenStore` must honor these guarantees.
authBase never stores raw refresh JWTs; only SHA-256 hashes of the full token
string are persisted via `domain.HashRefreshToken`.

## Logical schema

Each refresh session record contains at least:

| Field | Meaning |
|---|---|
| `token_id` / jti | Unique identifier of this refresh token |
| `family_id` | Stable session family shared across rotations |
| `user_id` | Subject that owns the family |
| `token_hash` | SHA-256 of the raw refresh JWT (32 bytes) |
| `issued_at` | Issue timestamp |
| `expires_at` | Absolute expiry |
| `consumed_at` | Set when the token is rotated away (nullable) |
| `revoked_family` | Family-level revocation marker (table or flag) |

Suggested uniqueness:

- unique index on `token_hash`
- unique index on `token_id`
- index on `family_id` for bulk revoke and cleanup

## Atomicity

`Create` inserts a new active session.

`Rotate(currentHash, next)` must, in a single transaction / compare-and-swap:

1. load the session for `currentHash`
2. reject if missing, expired, already consumed, or family revoked
3. mark the current session consumed
4. insert `next`
5. commit

Concurrent `Rotate` calls on the same hash: exactly one succeeds; the loser
receives `ErrRefreshConsumed` (or equivalent mapped by the adapter).

Detecting `ErrRefreshConsumed` is a replay signal: authBase will call
`RevokeFamily` and deny further use of that family.

`RevokeFamily` marks every session in the family unusable for future rotates.

## Retention

Expired and consumed sessions may be purged by the adapter on its own schedule.
Retention windows are an operational choice; authBase does not require indefinite
history beyond what is needed to detect recent replay.

## Issuance boundary

Only `AuthenticationService` mints refresh token pairs for callers. It always
`Create`s or `Rotate`s the session in `RefreshTokenStore` before returning
tokens. `TokenManager` pair issuance is unexported so adapters cannot return
refresh JWTs that were never registered.

## Non-goals

- Do not store plaintext JWTs, access tokens, or password hashes in this store.
- Do not implement rotation with best-effort dual writes; if the datastore cannot
  provide atomic consume+insert, do not adapt it for production refresh.
- Do not expose TokenManager helpers that return refresh tokens without a store
  write; that breaks replay detection.
