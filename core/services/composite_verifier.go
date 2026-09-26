package services

import (
	"github.com/WilsonSayago/authBase/v4/core/port"
	"github.com/golang-jwt/jwt/v5"
)

// CompositeVerifier tries each configured verifier in order. It is used for
// documented overlap windows, not for following remote key locators.
type CompositeVerifier struct {
	verifiers []port.TokenVerifier
}

func NewCompositeVerifier(verifiers ...port.TokenVerifier) (*CompositeVerifier, error) {
	active := make([]port.TokenVerifier, 0, len(verifiers))
	for _, verifier := range verifiers {
		if verifier != nil {
			active = append(active, verifier)
		}
	}
	if len(active) == 0 {
		return nil, errTokenVerifierRequired
	}
	return &CompositeVerifier{verifiers: active}, nil
}

func (v *CompositeVerifier) Parse(tokenString string, claims jwt.Claims, opts ...jwt.ParserOption) (*jwt.Token, error) {
	var last error
	for _, verifier := range v.verifiers {
		// Each attempt needs a fresh claims destination because a failed parse
		// may have partially populated the caller's struct.
		clone := cloneClaims(claims)
		token, err := verifier.Parse(tokenString, clone, opts...)
		if err == nil {
			copyClaims(claims, clone)
			return token, nil
		}
		last = err
	}
	return nil, last
}

func cloneClaims(claims jwt.Claims) jwt.Claims {
	switch c := claims.(type) {
	case *TokenClaims:
		cp := *c
		return &cp
	default:
		return claims
	}
}

func copyClaims(dst, src jwt.Claims) {
	to, ok := dst.(*TokenClaims)
	if !ok {
		return
	}
	from, ok := src.(*TokenClaims)
	if !ok {
		return
	}
	*to = *from
}

var errTokenVerifierRequired = errString("token verifier is required")

type errString string

func (e errString) Error() string { return string(e) }

var _ port.TokenVerifier = (*CompositeVerifier)(nil)
