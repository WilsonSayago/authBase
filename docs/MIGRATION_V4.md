# Migrating from authBase v3 to v4

v4 changes role-list pagination from offset/page metadata to forward-only
keyset pagination. Authentication, JWT claims, refresh rotation, and password
handling are unchanged from v3.

## Module path

Update the dependency and every import:

```sh
go get github.com/WilsonSayago/authBase/v4@v4.0.0
go mod tidy
```

```go
import (
    "github.com/WilsonSayago/authBase/v4/core/domain"
    "github.com/WilsonSayago/authBase/v4/core/port"
)
```

## Role listing contract

Before (v3), role listing used numeric page/size arguments and returned exact
totals. In v4:

```go
FindAll(ctx context.Context, request domain.PageRequest) (domain.Page[domain.Role], error)
GetRoles(ctx context.Context, request domain.PageRequest) (domain.Page[domain.Role], error)
```

`PageRequest` contains a positive `Limit` and an optional `After *CursorKey`.
`Page[T]` contains `Items` and `HasNext`; it deliberately has no page number or
exact total.

## Adapter requirements

1. Order rows by a stable, unique keyset such as `(created_at, id) DESC`.
2. When `After` is present, select rows strictly after that key in the chosen
   traversal direction.
3. Query `Limit + 1` rows, trim the extra row, and set `HasNext`.
4. Encode/decode cursors in the HTTP adapter. Cursors are transport concerns,
   not part of authBase.
5. Reject malformed cursors and invalid limits before calling the use case.

## Token compatibility

The v4 pagination change does not alter token claims or signing rules. A
v3 → v4 upgrade does not require session invalidation when issuer, audience,
and signing secrets remain unchanged.

## Consumer checklist

1. Replace `/v3` imports with `/v4`.
2. Update every `RolePort.FindAll` implementation.
3. Update calls to `RoleUseCase.GetRoles`.
4. Replace offset/page response fields with `hasNext` and an adapter-generated
   continuation cursor.
5. Test first page, next page, malformed cursor, empty result, and stable
   ordering when timestamps are equal.
6. Run `go test ./...`, `go test -race ./...`, and `go vet ./...`.
