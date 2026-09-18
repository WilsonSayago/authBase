package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/WilsonSayago/authBase/infra/config/properties"
	"github.com/golang-jwt/jwt/v5"
)

// TokenType distinguishes access and refresh tokens in a dedicated claim.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// TokenClaims is the typed JWT payload used by authBase.
type TokenClaims struct {
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

// TokenManager issues and verifies access/refresh tokens with a single policy.
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
		newID: newRandomJTI,
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

// IssuePair creates a signed access and refresh token for subject.
func (m *TokenManager) IssuePair(subject string) (accessToken, refreshToken string, err error) {
	if subject == "" {
		return "", "", fmt.Errorf("token subject must not be empty")
	}
	accessToken, err = m.issue(subject, TokenTypeAccess, m.cfg.SecretKey, m.cfg.ExpirationTime)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = m.issue(subject, TokenTypeRefresh, m.cfg.RefreshSecret, m.cfg.RefreshTokenTime)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// ParseAccess verifies an access token and returns typed claims.
func (m *TokenManager) ParseAccess(tokenString string) (*TokenClaims, error) {
	return m.parse(tokenString, TokenTypeAccess, m.cfg.SecretKey)
}

// ParseRefresh verifies a refresh token and returns typed claims.
func (m *TokenManager) ParseRefresh(tokenString string) (*TokenClaims, error) {
	return m.parse(tokenString, TokenTypeRefresh, m.cfg.RefreshSecret)
}

func (m *TokenManager) issue(subject string, tokenType TokenType, secret string, lifetimeHours int) (string, error) {
	jti, err := m.newID()
	if err != nil {
		return "", fmt.Errorf("generate token id: %w", err)
	}
	now := m.now()
	claims := TokenClaims{
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ID:        jti,
			Issuer:    m.cfg.Issuer,
			Audience:  []string{m.cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(lifetimeHours) * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
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
	return claims, nil
}

func newRandomJTI() (string, error) {
	buf := make([]byte, 16) // 128 bits
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
