package core

import domain "github.com/WilsonSayago/authBase/core/domain"

type AuthenticationUseCase interface {
	GetToken(id string) (string, string, error)
	Login(username, password string) (string, string, error)
	RefreshToken(refreshToken string) (string, string, error)
	ValidateToken(tokenString string) (domain.IUserGeneric, error)
	ValidateTokenAndRefresh() (string, error)
}
