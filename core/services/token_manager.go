package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/WilsonSayago/authBase/v4/core/domain"
	"github.com/WilsonSayago/authBase/v4/core/port"
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

// TokenCrypto is the explicit signer/verifier pair for one TokenManager.
type TokenCrypto struct {
	AccessSigner    port.TokenSigner
	RefreshSigner   port.TokenSigner
	AccessVerifier  port.TokenVerifier
	RefreshVerifier port.TokenVerifier
}

// TokenManager verifies access/refresh tokens and privately issues pairs for
// AuthenticationService. Callers outside this package must not mint refresh
// tokens; only AuthenticationService persists sessions before returning them.
type TokenManager struct {
	cfg    properties.Jwt
	now    func() time.Time
	newID  func() (string, error)
	crypto TokenCrypto
}

// NewTokenManager validates cfg and returns an independent HMAC token manager.
// Issued tokens remain HS256. New tokens set typ; tokens without kid remain
// acceptable so existing HMAC material continues to verify.
func NewTokenManager(cfg properties.Jwt) (*TokenManager, error) {
	crypto, err := NewHMACCrypto(cfg, nil, nil)
	if err != nil {
		return nil, err
	}
	return NewTokenManagerWithCrypto(cfg, crypto)
}

// NewHMACCrypto builds the default HS256 signer/verifier pair.
// previousAccess/previousRefresh may be nil. Empty kids keep legacy verify.
func NewHMACCrypto(cfg properties.Jwt, previousAccess, previousRefresh []HMACKey) (TokenCrypto, error) {
	if err := cfg.Validate(); err != nil {
		return TokenCrypto{}, err
	}
	return NewHMACCryptoWithKeys(
		HMACKey{Secret: []byte(cfg.SecretKey)},
		HMACKey{Secret: []byte(cfg.RefreshSecret)},
		previousAccess,
		previousRefresh,
	)
}

// NewHMACCryptoWithKeys builds HS256 adapters from explicit key material.
// Empty kids keep legacy verify for tokens minted without kid.
func NewHMACCryptoWithKeys(access, refresh HMACKey, previousAccess, previousRefresh []HMACKey) (TokenCrypto, error) {
	if len(access.Secret) == 0 || len(refresh.Secret) == 0 {
		return TokenCrypto{}, fmt.Errorf("hmac secret is required")
	}
	if access.ID != "" && access.ID == refresh.ID {
		return TokenCrypto{}, fmt.Errorf("access and refresh kids must be different")
	}
	accessSigner, err := newHMACSigner(TokenTypeAccess, access)
	if err != nil {
		return TokenCrypto{}, err
	}
	refreshSigner, err := newHMACSigner(TokenTypeRefresh, refresh)
	if err != nil {
		return TokenCrypto{}, err
	}
	accessVerifier, err := newHMACVerifier(TokenTypeAccess, access, previousAccess, true)
	if err != nil {
		return TokenCrypto{}, err
	}
	refreshVerifier, err := newHMACVerifier(TokenTypeRefresh, refresh, previousRefresh, true)
	if err != nil {
		return TokenCrypto{}, err
	}
	return TokenCrypto{
		AccessSigner:    accessSigner,
		RefreshSigner:   refreshSigner,
		AccessVerifier:  accessVerifier,
		RefreshVerifier: refreshVerifier,
	}, nil
}

// NewEd25519Crypto builds an EdDSA signer/verifier pair. Access and refresh
// keys must be distinct. Previous keys are verification-only.
func NewEd25519Crypto(access, refresh Ed25519Key, previousAccess, previousRefresh []Ed25519Key) (TokenCrypto, error) {
	if access.ID == "" || refresh.ID == "" {
		return TokenCrypto{}, fmt.Errorf("access and refresh kids are required")
	}
	if access.ID == refresh.ID {
		return TokenCrypto{}, fmt.Errorf("access and refresh kids must be different")
	}
	accessSigner, err := newEd25519Signer(TokenTypeAccess, access)
	if err != nil {
		return TokenCrypto{}, err
	}
	refreshSigner, err := newEd25519Signer(TokenTypeRefresh, refresh)
	if err != nil {
		return TokenCrypto{}, err
	}
	accessVerifier, err := newEd25519Verifier(TokenTypeAccess, access, previousAccess)
	if err != nil {
		return TokenCrypto{}, err
	}
	refreshVerifier, err := newEd25519Verifier(TokenTypeRefresh, refresh, previousRefresh)
	if err != nil {
		return TokenCrypto{}, err
	}
	return TokenCrypto{
		AccessSigner:    accessSigner,
		RefreshSigner:   refreshSigner,
		AccessVerifier:  accessVerifier,
		RefreshVerifier: refreshVerifier,
	}, nil
}

// NewTokenManagerWithCrypto constructs a manager with explicit crypto adapters.
func NewTokenManagerWithCrypto(cfg properties.Jwt, crypto TokenCrypto) (*TokenManager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if crypto.AccessSigner == nil || crypto.RefreshSigner == nil || crypto.AccessVerifier == nil || crypto.RefreshVerifier == nil {
		return nil, fmt.Errorf("token signer and verifier are required")
	}
	return &TokenManager{
		cfg:    cfg,
		now:    time.Now,
		newID:  newRandomID,
		crypto: crypto,
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
	return m.parse(tokenString, TokenTypeAccess, m.crypto.AccessVerifier)
}

// ParseRefresh verifies a refresh token and returns typed claims.
func (m *TokenManager) ParseRefresh(tokenString string) (*TokenClaims, error) {
	return m.parse(tokenString, TokenTypeRefresh, m.crypto.RefreshVerifier)
}

func (m *TokenManager) issuePair(subject, familyID string) (issuedTokens, error) {
	if subject == "" {
		return issuedTokens{}, fmt.Errorf("token subject must not be empty")
	}
	accessToken, _, err := m.issue(subject, TokenTypeAccess, "", m.crypto.AccessSigner, m.cfg.ExpirationTime)
	if err != nil {
		return issuedTokens{}, err
	}
	refreshToken, session, err := m.issue(subject, TokenTypeRefresh, familyID, m.crypto.RefreshSigner, m.cfg.RefreshTokenTime)
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
	signer port.TokenSigner,
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
	signed, err := signer.Sign(claims)
	if err != nil {
		return "", domain.RefreshSession{}, err
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

func (m *TokenManager) parse(tokenString string, expectedType TokenType, verifier port.TokenVerifier) (*TokenClaims, error) {
	claims := &TokenClaims{}
	token, err := verifier.Parse(tokenString, claims, parseOptions(m.cfg, m.now)...)
	if err != nil {
		return nil, err
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
