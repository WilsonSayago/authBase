package port

import (
	"context"

	domain "github.com/WilsonSayago/authBase/core/domain"
)

// UserReader loads complete user profiles for authorization and token validation.
type UserReader[T domain.IUserGeneric] interface {
	FindByID(ctx context.Context, id string) (T, error)
}
