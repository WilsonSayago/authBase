package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/WilsonSayago/authBase/core"
	domain "github.com/WilsonSayago/authBase/core/domain"
	"github.com/WilsonSayago/authBase/core/port"
	"github.com/WilsonSayago/authBase/infra/config/properties"
)

type AuthenticationService[T domain.IUserGeneric] struct {
	users        port.UserReader[T]
	credentials  port.CredentialReader
	validatePort port.ValidationPort
	tokens       *TokenManager
}

// NewAuthenticationService constructs an authentication service with validated JWT configuration.
func NewAuthenticationService[T domain.IUserGeneric](
	users port.UserReader[T],
	credentials port.CredentialReader,
	validatePort port.ValidationPort,
	cfg properties.Jwt,
) (*AuthenticationService[T], error) {
	if users == nil {
		return nil, fmt.Errorf("authentication user reader is nil")
	}
	if credentials == nil {
		return nil, fmt.Errorf("authentication credential reader is nil")
	}
	if validatePort == nil {
		return nil, fmt.Errorf("authentication validation port is nil")
	}
	tokens, err := NewTokenManager(cfg)
	if err != nil {
		return nil, err
	}
	return &AuthenticationService[T]{
		users:        users,
		credentials:  credentials,
		validatePort: validatePort,
		tokens:       tokens,
	}, nil
}

// GetAuthenticationInstance is deprecated; prefer NewAuthenticationService.
//
// Deprecated: use NewAuthenticationService.
func GetAuthenticationInstance[T domain.IUserGeneric](
	users port.UserReader[T],
	credentials port.CredentialReader,
	validatePort port.ValidationPort,
	prop *properties.JwtProp,
) (core.AuthenticationUseCase, error) {
	if prop == nil {
		return nil, fmt.Errorf("jwt configuration is nil")
	}
	return NewAuthenticationService[T](users, credentials, validatePort, prop.Jwt)
}

func (a AuthenticationService[T]) GetToken(id string) (string, string, error) {
	return a.tokens.IssuePair(id)
}

func (a AuthenticationService[T]) Login(ctx context.Context, username, password string) (string, string, error) {
	cred, err := a.credentials.FindCredentialsByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return "", "", core.ErrInvalidCredentials
		}
		return "", "", err
	}
	if !cred.Active {
		return "", "", core.ErrInvalidCredentials
	}
	if !a.validatePort.CheckPassword(cred.PasswordHash, password) {
		return "", "", core.ErrInvalidCredentials
	}

	token, refreshToken, err := a.GetToken(cred.UserID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate token: %w", err)
	}
	return token, refreshToken, nil
}

func (a AuthenticationService[T]) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := a.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid token")
	}

	user, err := a.requireActiveUser(ctx, claims.Subject)
	if err != nil {
		return "", "", err
	}

	tokenString, newRefresh, err := a.GetToken(user.GetId())
	if err != nil {
		return "", "", fmt.Errorf("failed to generate token: %w", err)
	}
	return tokenString, newRefresh, nil
}

func (a AuthenticationService[T]) ValidateToken(ctx context.Context, tokenString string) (domain.IUserGeneric, error) {
	claims, err := a.tokens.ParseAccess(tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	user, err := a.requireActiveUser(ctx, claims.Subject)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (a AuthenticationService[T]) requireActiveUser(ctx context.Context, id string) (T, error) {
	var zero T
	user, err := a.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return zero, core.ErrNotFound
		}
		return zero, err
	}
	if !user.GetActive() {
		return zero, core.ErrInactiveIdentity
	}
	return user, nil
}
