package domain

import "time"

// CursorKey identifies one item in a stable keyset ordering.
type CursorKey struct {
	CreatedAt time.Time
	ID        string
}

// PageRequest defines a forward-only keyset page.
type PageRequest struct {
	Limit int
	After *CursorKey
}

// Page contains one keyset page and its continuation state.
type Page[T any] struct {
	Items   []T
	HasNext bool
}
