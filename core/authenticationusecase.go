package core

import domain "github.com/WilsonSayago/authBase/core/domains"

type AuthenticationUseCase interface {
	GetToken(id string) (string, error)
	Login(username, password string) (string, error)
	RefreshToken() (string, error)
	ValidateToken(tokenString string) (domain.IUserGeneric, error)
	ValidateTokenAndRefresh() (string, error)
}
