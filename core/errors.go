package core

import "errors"

var (
	// ErrInvalidCredentials is the public login failure for missing user,
	// wrong password, or inactive account. Callers must not distinguish them.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrInactiveIdentity is returned when a previously issued token belongs
	// to a user that is no longer active.
	ErrInactiveIdentity = errors.New("inactive identity")

	// ErrNotFound is the sentinel repositories should return when a user or
	// credential record does not exist.
	ErrNotFound = errors.New("not found")

	// ErrUnavailable is the sentinel for infrastructure failures in identity stores.
	ErrUnavailable = errors.New("identity store unavailable")
)
