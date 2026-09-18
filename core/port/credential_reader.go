package port

import "context"

// CredentialRecord holds authentication material without exposing it via IUserGeneric.
type CredentialRecord struct {
	UserID       string
	Username     string
	PasswordHash string
	Active       bool
}

// CredentialReader loads credential records by username/email for login.
type CredentialReader interface {
	FindCredentialsByUsername(ctx context.Context, username string) (CredentialRecord, error)
}
