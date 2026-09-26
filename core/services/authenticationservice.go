package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/WilsonSayago/authBase/v4/core"
	domain "github.com/WilsonSayago/authBase/v4/core/domain"
	"github.com/WilsonSayago/authBase/v4/core/port"
	"github.com/WilsonSayago/authBase/v4/infra/config/properties"
)

type AuthenticationService[T domain.IUserGeneric] struct {
	users        port.UserReader[T]
	credentials  port.CredentialReader
	refreshStore port.RefreshTokenStore
	validatePort port.ValidationPort
	tokens       *TokenManager
	dummyHash    string
	events       core.SecurityEventSink
}

const dummyPassword = "authbase-dummy-password-never-use"

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
	return newAuthenticationService(users, credentials, refreshStore, validatePort, tokens)
}

// NewAuthenticationServiceWithCrypto uses an explicit signer/verifier pair.
func NewAuthenticationServiceWithCrypto[T domain.IUserGeneric](
	users port.UserReader[T],
	credentials port.CredentialReader,
	refreshStore port.RefreshTokenStore,
	validatePort port.ValidationPort,
	cfg properties.Jwt,
	crypto TokenCrypto,
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
	tokens, err := NewTokenManagerWithCrypto(cfg, crypto)
	if err != nil {
		return nil, err
	}
	return newAuthenticationService(users, credentials, refreshStore, validatePort, tokens)
}

func newAuthenticationService[T domain.IUserGeneric](
	users port.UserReader[T],
	credentials port.CredentialReader,
	refreshStore port.RefreshTokenStore,
	validatePort port.ValidationPort,
	tokens *TokenManager,
) (*AuthenticationService[T], error) {
	dummyHash, err := validatePort.HashPassword(dummyPassword)
	if err != nil {
		return nil, fmt.Errorf("create dummy credential: %w", err)
	}
	return &AuthenticationService[T]{
		users:        users,
		credentials:  credentials,
		refreshStore: refreshStore,
		validatePort: validatePort,
		tokens:       tokens,
		dummyHash:    dummyHash,
		events:       core.NoopSecurityEventSink{},
	}, nil
}

// WithSecurityEvents attaches an optional sink. Sink failures never fail auth.
func (a *AuthenticationService[T]) WithSecurityEvents(sink core.SecurityEventSink) *AuthenticationService[T] {
	if sink == nil {
		a.events = core.NoopSecurityEventSink{}
		return a
	}
	a.events = sink
	return a
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
	foundActive := err == nil && cred.Active
	if err != nil {
		if !errors.Is(err, core.ErrNotFound) {
			return "", "", err
		}
	}

	hash := a.dummyHash
	if foundActive {
		hash = cred.PasswordHash
	}
	passwordMatches := a.validatePort.CheckPassword(hash, password)
	if !foundActive || !passwordMatches {
		a.record(ctx, core.SecurityEvent{Type: core.SecurityLoginFailure, Outcome: "failure"})
		return "", "", core.ErrInvalidCredentials
	}
	access, refresh, err := a.issueAndPersist(ctx, cred.UserID)
	if err != nil {
		a.record(ctx, core.SecurityEvent{Type: core.SecurityLoginFailure, ActorID: cred.UserID, Outcome: "failure"})
		return "", "", err
	}
	a.record(ctx, core.SecurityEvent{Type: core.SecurityLoginSuccess, ActorID: cred.UserID, Outcome: "success"})
	return access, refresh, nil
}

// EstablishSession creates a persisted token session for an identity that the
// caller has already authenticated. This privileged boundary always verifies
// that the identity still exists and is active before issuing tokens.
func (a AuthenticationService[T]) EstablishSession(ctx context.Context, userID string) (string, string, error) {
	if userID == "" {
		return "", "", fmt.Errorf("user id must not be empty")
	}
	user, err := a.requireActiveUser(ctx, userID)
	if err != nil {
		return "", "", err
	}
	return a.issueAndPersist(ctx, user.GetId())
}

// RevokeUserSessions invalidates every refresh session owned by a user.
func (a AuthenticationService[T]) RevokeUserSessions(ctx context.Context, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if userID == "" {
		return fmt.Errorf("user id must not be empty")
	}
	revoker, ok := a.refreshStore.(port.UserSessionRevoker)
	if !ok {
		return core.ErrUserSessionRevocationUnsupported
	}
	if err := revoker.RevokeUser(ctx, userID); err != nil {
		return fmt.Errorf("revoke user sessions: %w", err)
	}
	a.record(ctx, core.SecurityEvent{Type: core.SecurityUserRevoke, ActorID: userID, Outcome: "success"})
	return nil
}

func (a AuthenticationService[T]) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := a.tokens.ParseRefresh(refreshToken)
	if err != nil {
		a.record(ctx, core.SecurityEvent{Type: core.SecurityTokenReject, Outcome: "failure"})
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
	issued, err := a.tokens.issueRotatedPair(user.GetId(), claims.FamilyID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate token: %w", err)
	}

	if err := a.refreshStore.Rotate(ctx, currentHash, issued.Session); err != nil {
		if errors.Is(err, core.ErrRefreshConsumed) {
			if revokeErr := a.refreshStore.RevokeFamily(ctx, claims.FamilyID); revokeErr != nil {
				return "", "", errors.Join(core.ErrInvalidRefresh, err, fmt.Errorf("revoke family: %w", revokeErr))
			}
			a.record(ctx, core.SecurityEvent{
				Type: core.SecurityRefreshReplay, ActorID: claims.Subject, FamilyID: claims.FamilyID, Outcome: "failure",
			})
			return "", "", core.ErrInvalidRefresh
		}
		if errors.Is(err, core.ErrRefreshNotFound) || errors.Is(err, core.ErrRefreshFamilyRevoked) {
			return "", "", core.ErrInvalidRefresh
		}
		return "", "", err
	}

	a.record(ctx, core.SecurityEvent{
		Type: core.SecurityRefreshSuccess, ActorID: user.GetId(), FamilyID: claims.FamilyID, TokenID: claims.ID, Outcome: "success",
	})
	return issued.AccessToken, issued.RefreshToken, nil
}

// RevokeRefreshFamily invalidates every refresh session in the family.
func (a AuthenticationService[T]) RevokeRefreshFamily(ctx context.Context, familyID string) error {
	if familyID == "" {
		return fmt.Errorf("refresh family id must not be empty")
	}
	if err := a.refreshStore.RevokeFamily(ctx, familyID); err != nil {
		return err
	}
	a.record(ctx, core.SecurityEvent{Type: core.SecurityFamilyRevoke, FamilyID: familyID, Outcome: "success"})
	return nil
}

func (a AuthenticationService[T]) ValidateToken(ctx context.Context, tokenString string) (domain.IUserGeneric, error) {
	claims, err := a.tokens.ParseAccess(tokenString)
	if err != nil {
		a.record(ctx, core.SecurityEvent{Type: core.SecurityTokenReject, Outcome: "failure"})
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	user, err := a.requireActiveUser(ctx, claims.Subject)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (a AuthenticationService[T]) issueAndPersist(ctx context.Context, userID string) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	issued, err := a.tokens.issueInitialPair(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate token: %w", err)
	}
	if err := a.refreshStore.Create(ctx, issued.Session); err != nil {
		return "", "", fmt.Errorf("persist refresh session: %w", err)
	}
	return issued.AccessToken, issued.RefreshToken, nil
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

func (a AuthenticationService[T]) record(ctx context.Context, event core.SecurityEvent) {
	if a.events == nil {
		return
	}
	defer func() { _ = recover() }()
	a.events.Record(ctx, event)
}
