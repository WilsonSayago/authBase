package core

import (
	"context"

	domain "github.com/WilsonSayago/authBase/v3/core/domain"
)

type AuthenticationUseCase interface {
	Login(ctx context.Context, username, password string) (string, string, error)
	RefreshToken(ctx context.Context, refreshToken string) (string, string, error)
	ValidateToken(ctx context.Context, tokenString string) (domain.IUserGeneric, error)
	RevokeRefreshFamily(ctx context.Context, familyID string) error
}
