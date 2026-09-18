package core

import (
	"context"

	domain "github.com/WilsonSayago/authBase/core/domain"
)

type AuthenticationUseCase interface {
	GetToken(id string) (string, string, error)
	Login(ctx context.Context, username, password string) (string, string, error)
	RefreshToken(ctx context.Context, refreshToken string) (string, string, error)
	ValidateToken(ctx context.Context, tokenString string) (domain.IUserGeneric, error)
}
