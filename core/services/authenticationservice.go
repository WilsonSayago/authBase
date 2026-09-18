package services

import (
	"fmt"

	"github.com/WilsonSayago/authBase/core"
	domain "github.com/WilsonSayago/authBase/core/domain"
	"github.com/WilsonSayago/authBase/core/port"
	"github.com/WilsonSayago/authBase/infra/config/properties"
)

type AuthenticationService[T domain.IUserGeneric] struct {
	port         port.GenericPort[T]
	validatePort port.ValidationPort
	tokens       *TokenManager
}

// NewAuthenticationService constructs an authentication service with validated JWT configuration.
func NewAuthenticationService[T domain.IUserGeneric](
	userPort port.GenericPort[T],
	validatePort port.ValidationPort,
	cfg properties.Jwt,
) (*AuthenticationService[T], error) {
	if userPort == nil {
		return nil, fmt.Errorf("authentication user port is nil")
	}
	if validatePort == nil {
		return nil, fmt.Errorf("authentication validation port is nil")
	}
	tokens, err := NewTokenManager(cfg)
	if err != nil {
		return nil, err
	}
	return &AuthenticationService[T]{
		port:         userPort,
		validatePort: validatePort,
		tokens:       tokens,
	}, nil
}

// GetAuthenticationInstance is deprecated; prefer NewAuthenticationService.
//
// Deprecated: use NewAuthenticationService.
func GetAuthenticationInstance[T domain.IUserGeneric](
	userPort port.GenericPort[T],
	validatePort port.ValidationPort,
	prop *properties.JwtProp,
) (core.AuthenticationUseCase, error) {
	if prop == nil {
		return nil, fmt.Errorf("jwt configuration is nil")
	}
	return NewAuthenticationService[T](userPort, validatePort, prop.Jwt)
}

func (a AuthenticationService[T]) GetToken(id string) (string, string, error) {
	return a.tokens.IssuePair(id)
}

func (a AuthenticationService[T]) Login(username, password string) (string, string, error) {
	user, err := a.port.FindByEmail(username)
	if err != nil {
		return "", "", fmt.Errorf("user not found: %w", err)
	}

	if !a.validatePort.CheckPassword(user.GetPassword(), password) {
		return "", "", fmt.Errorf("invalid password")
	}

	token, refreshToken, err := a.GetToken(user.GetId())
	if err != nil {
		return "", "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, refreshToken, nil
}

func (a AuthenticationService[T]) RefreshToken(refreshToken string) (string, string, error) {
	claims, err := a.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid token")
	}

	tokenString, newRefresh, err := a.GetToken(claims.Subject)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, newRefresh, nil
}

func (a AuthenticationService[T]) ValidateToken(tokenString string) (domain.IUserGeneric, error) {
	claims, err := a.tokens.ParseAccess(tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	user, err := a.port.FindFullById(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}
