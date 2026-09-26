package port

import "github.com/golang-jwt/jwt/v5"

// TokenSigner creates a compact JWT without exposing key material.
type TokenSigner interface {
	Sign(claims jwt.Claims) (string, error)
}

// TokenVerifier parses and validates a compact JWT against locally configured keys.
// Implementations must not follow jku, x5u, or other remote key locators from the token.
type TokenVerifier interface {
	Parse(tokenString string, claims jwt.Claims, opts ...jwt.ParserOption) (*jwt.Token, error)
}
