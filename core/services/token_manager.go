package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/WilsonSayago/authBase/v4/core/domain"
	"github.com/WilsonSayago/authBase/v4/infra/config/properties"
	"github.com/golang-jwt/jwt/v5"
)

// TokenType distinguishes access and refresh tokens in a dedicated claim.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// TokenClaims is the typed JWT payload used by authBase.
// FamilyID is set only on refresh tokens.
type TokenClaims struct {
	TokenType TokenType `json:"token_type"`
	FamilyID  string    `json:"family_id,omitempty"`
	jwt.RegisteredClaims
}

// issuedTokens is a signed access/refresh pair plus the refresh session metadata
// that AuthenticationService must persist before returning tokens to a client.
type issuedTokens struct {
	AccessToken  string
	RefreshToken string
	Session      domain.RefreshSession
}

// TokenManager verifies access/refresh tokens and privately issues pairs for
// AuthenticationService. Callers outside this package must not mint refresh
// tokens; only AuthenticationService persists sessions before returning them.
type TokenManager struct {
	cfg   properties.Jwt
	now   func() time.Time
	newID func() (string, error)
}

// NewTokenManager validates cfg and returns an independent token manager.
func NewTokenManager(cfg properties.Jwt) (*TokenManager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &TokenManager{
		cfg:   cfg,
		now:   time.Now,
		newID: newRandomID,
	}, nil
}

func newTokenManagerForTest(cfg properties.Jwt, now func() time.Time, newID func() (string, error)) (*TokenManager, error) {
	tm, err := NewTokenManager(cfg)
	if err != nil {
		return nil, err
	}
	if now != nil {
		tm.now = now
	}
	if newID != nil {
		tm.newID = newID
	}
	return tm, nil
}

// issueInitialPair creates access/refresh tokens with a new refresh family.
// Only AuthenticationService should call this, and must Create the session first.
func (m *TokenManager) issueInitialPair(subject string) (issuedTokens, error) {
	familyID, err := m.newID()
	if err != nil {
		return issuedTokens{}, fmt.Errorf("generate family id: %w", err)
	}
	return m.issuePair(subject, familyID)
}

// issueRotatedPair creates access/refresh tokens that continue an existing family.
// Only AuthenticationService should call this, and must Rotate before returning.
func (m *TokenManager) issueRotatedPair(subject, familyID string) (issuedTokens, error) {
	if familyID == "" {
		return issuedTokens{}, fmt.Errorf("refresh family id must not be empty")
	}
	return m.issuePair(subject, familyID)
}

// ParseAccess verifies an access token and returns typed claims.
func (m *TokenManager) ParseAccess(tokenString string) (*TokenClaims, error) {
	return m.parse(tokenString, TokenTypeAccess, m.cfg.SecretKey)
}

// ParseRefresh verifies a refresh token and returns typed claims.
func (m *TokenManager) ParseRefresh(tokenString string) (*TokenClaims, error) {
	return m.parse(tokenString, TokenTypeRefresh, m.cfg.RefreshSecret)
}

func (m *TokenManager) issuePair(subject, familyID string) (issuedTokens, error) {
	if subject == "" {
		return issuedTokens{}, fmt.Errorf("token subject must not be empty")
	}
	accessToken, _, err := m.issue(subject, TokenTypeAccess, "", m.cfg.SecretKey, m.cfg.ExpirationTime)
	if err != nil {
		return issuedTokens{}, err
	}
	refreshToken, session, err := m.issue(subject, TokenTypeRefresh, familyID, m.cfg.RefreshSecret, m.cfg.RefreshTokenTime)
	if err != nil {
		return issuedTokens{}, err
	}
	return issuedTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Session:      session,
	}, nil
}

func (m *TokenManager) issue(
	subject string,
	tokenType TokenType,
	familyID string,
	secret string,
	lifetimeHours int,
) (string, domain.RefreshSession, error) {
	jti, err := m.newID()
	if err != nil {
		return "", domain.RefreshSession{}, fmt.Errorf("generate token id: %w", err)
	}
	now := m.now()
	expiresAt := now.Add(time.Duration(lifetimeHours) * time.Hour)
	claims := TokenClaims{
		TokenType: tokenType,
		FamilyID:  familyID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ID:        jti,
			Issuer:    m.cfg.Issuer,
			Audience:  []string{m.cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", domain.RefreshSession{}, fmt.Errorf("sign token: %w", err)
	}

	session := domain.RefreshSession{}
	if tokenType == TokenTypeRefresh {
		session = domain.RefreshSession{
			TokenID:   jti,
			FamilyID:  familyID,
			UserID:    subject,
			TokenHash: domain.HashRefreshToken(signed),
			IssuedAt:  now,
			ExpiresAt: expiresAt,
		}
	}
	return signed, session, nil
}

func (m *TokenManager) parse(tokenString string, expectedType TokenType, secret string) (*TokenClaims, error) {
	claims := &TokenClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(m.cfg.Issuer),
		jwt.WithAudience(m.cfg.Audience),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(time.Duration(m.cfg.LeewaySeconds)*time.Second),
		jwt.WithStrictDecoding(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.TokenType != expectedType {
		return nil, fmt.Errorf("invalid token type")
	}
	if claims.IssuedAt == nil {
		return nil, fmt.Errorf("token issued-at is required")
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("token subject must not be empty")
	}
	if claims.ID == "" {
		return nil, fmt.Errorf("token id must not be empty")
	}
	if expectedType == TokenTypeAccess && claims.FamilyID != "" {
		return nil, fmt.Errorf("access token must not include family id")
	}
	if expectedType == TokenTypeRefresh && claims.FamilyID == "" {
		return nil, fmt.Errorf("refresh family id is required")
	}
	return claims, nil
}

func newRandomID() (string, error) {
	buf := make([]byte, 16) // 128 bits
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
