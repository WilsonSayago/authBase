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
	refreshStore port.RefreshTokenStore
	validatePort port.ValidationPort
	tokens       *TokenManager
}

// NewAuthenticationService constructs an authentication service with validated JWT configuration.
func NewAuthenticationService[T domain.IUserGeneric](
	users port.UserReader[T],
	credentials port.CredentialReader,
	refreshStore port.RefreshTokenStore,
	validatePort port.ValidationPort,
	cfg properties.Jwt,
) (*AuthenticationService[T], error) {
	if users == nil {
		return nil, fmt.Errorf("authentication user reader is nil")
	}
	if credentials == nil {
		return nil, fmt.Errorf("authentication credential reader is nil")
	}
	if refreshStore == nil {
		return nil, fmt.Errorf("authentication refresh store is nil")
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
		refreshStore: refreshStore,
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
	refreshStore port.RefreshTokenStore,
	validatePort port.ValidationPort,
	prop *properties.JwtProp,
) (core.AuthenticationUseCase, error) {
	if prop == nil {
		return nil, fmt.Errorf("jwt configuration is nil")
	}
	return NewAuthenticationService[T](users, credentials, refreshStore, validatePort, prop.Jwt)
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

	issued, err := a.tokens.IssueInitialPair(cred.UserID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate token: %w", err)
	}
	if err := a.refreshStore.Create(ctx, issued.Session); err != nil {
		return "", "", fmt.Errorf("persist refresh session: %w", err)
	}
	return issued.AccessToken, issued.RefreshToken, nil
}

func (a AuthenticationService[T]) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := a.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return "", "", core.ErrInvalidRefresh
	}

	user, err := a.requireActiveUser(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, core.ErrInactiveIdentity) || errors.Is(err, core.ErrNotFound) {
			return "", "", core.ErrInvalidRefresh
		}
		return "", "", err
	}

	currentHash := domain.HashRefreshToken(refreshToken)
	issued, err := a.tokens.IssueRotatedPair(user.GetId(), claims.FamilyID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate token: %w", err)
	}

	if err := a.refreshStore.Rotate(ctx, currentHash, issued.Session); err != nil {
		if errors.Is(err, core.ErrRefreshConsumed) {
			if revokeErr := a.refreshStore.RevokeFamily(ctx, claims.FamilyID); revokeErr != nil {
				return "", "", errors.Join(core.ErrInvalidRefresh, err, fmt.Errorf("revoke family: %w", revokeErr))
			}
			return "", "", core.ErrInvalidRefresh
		}
		if errors.Is(err, core.ErrRefreshNotFound) || errors.Is(err, core.ErrRefreshFamilyRevoked) {
			return "", "", core.ErrInvalidRefresh
		}
		return "", "", err
	}

	return issued.AccessToken, issued.RefreshToken, nil
}

// RevokeRefreshFamily invalidates every refresh session in the family.
func (a AuthenticationService[T]) RevokeRefreshFamily(ctx context.Context, familyID string) error {
	if familyID == "" {
		return fmt.Errorf("refresh family id must not be empty")
	}
	return a.refreshStore.RevokeFamily(ctx, familyID)
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
