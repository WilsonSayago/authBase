package services

import (
	"fmt"

	"github.com/WilsonSayago/authBase/v4/core/port"
	"github.com/golang-jwt/jwt/v5"
)

// HMACKey is a locally configured HS256 secret with an optional kid.
type HMACKey struct {
	ID     string
	Secret []byte
}

type hmacSigner struct {
	typ string
	key HMACKey
}

func newHMACSigner(tokenType TokenType, key HMACKey) (*hmacSigner, error) {
	if len(key.Secret) == 0 {
		return nil, fmt.Errorf("hmac secret is required")
	}
	return &hmacSigner{typ: typFor(tokenType), key: key}, nil
}

func (s *hmacSigner) Sign(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["typ"] = s.typ
	if s.key.ID != "" {
		token.Header["kid"] = s.key.ID
	}
	signed, err := token.SignedString(s.key.Secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

type hmacVerifier struct {
	expectedType TokenType
	expectedTyp  string
	allowLegacy  bool
	keys         map[string][]byte
	legacySecret []byte
}

func newHMACVerifier(tokenType TokenType, active HMACKey, previous []HMACKey, allowLegacy bool) (*hmacVerifier, error) {
	if len(active.Secret) == 0 {
		return nil, fmt.Errorf("hmac secret is required")
	}
	keys := map[string][]byte{}
	if active.ID != "" {
		keys[active.ID] = active.Secret
	}
	for _, key := range previous {
		if key.ID == "" || len(key.Secret) == 0 {
			return nil, fmt.Errorf("previous hmac key must include kid and secret")
		}
		if _, exists := keys[key.ID]; exists {
			return nil, fmt.Errorf("duplicate jwt kid")
		}
		keys[key.ID] = key.Secret
	}
	return &hmacVerifier{
		expectedType: tokenType,
		expectedTyp:  typFor(tokenType),
		allowLegacy:  allowLegacy,
		keys:         keys,
		legacySecret: active.Secret,
	}, nil
}

func (v *hmacVerifier) Parse(tokenString string, claims jwt.Claims, opts ...jwt.ParserOption) (*jwt.Token, error) {
	return parseSignedToken(tokenString, claims, []string{jwt.SigningMethodHS256.Alg()}, v.keyFunc, v.expectedTyp, v.allowLegacy, opts...)
}

func (v *hmacVerifier) keyFunc(token *jwt.Token) (any, error) {
	if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
		return nil, fmt.Errorf("unexpected jwt algorithm")
	}
	if _, ok := lookupRemoteKeyLocator(token.Header); ok {
		return nil, fmt.Errorf("remote jwt key locators are not allowed")
	}
	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		if !v.allowLegacy {
			return nil, fmt.Errorf("jwt kid is required")
		}
		return v.legacySecret, nil
	}
	secret, ok := v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("unknown jwt kid")
	}
	return secret, nil
}

func lookupRemoteKeyLocator(header map[string]any) (string, bool) {
	for _, name := range []string{"jku", "x5u", "jwk", "x5c"} {
		if _, ok := header[name]; ok {
			return name, true
		}
	}
	return "", false
}

var (
	_ port.TokenSigner   = (*hmacSigner)(nil)
	_ port.TokenVerifier = (*hmacVerifier)(nil)
)
