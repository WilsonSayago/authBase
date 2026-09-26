package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/WilsonSayago/authBase/v4/infra/config/properties"
	"github.com/golang-jwt/jwt/v5"
)

func parseSignedToken(
	tokenString string,
	claims jwt.Claims,
	methods []string,
	keyFunc jwt.Keyfunc,
	expectedTyp string,
	allowLegacyTyp bool,
	opts ...jwt.ParserOption,
) (*jwt.Token, error) {
	all := append([]jwt.ParserOption{jwt.WithValidMethods(methods)}, opts...)
	token, err := jwt.ParseWithClaims(tokenString, claims, keyFunc, all...)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if err := validateTyp(token, expectedTyp, allowLegacyTyp); err != nil {
		return nil, err
	}
	return token, nil
}

func validateTyp(token *jwt.Token, expectedTyp string, allowLegacy bool) error {
	raw, _ := token.Header["typ"].(string)
	typ := strings.ToLower(strings.TrimSpace(raw))
	if typ == strings.ToLower(expectedTyp) {
		return nil
	}
	if allowLegacy && (typ == "" || typ == strings.ToLower(tokenTypLegacy)) {
		return nil
	}
	return fmt.Errorf("invalid token typ")
}

func parseOptions(cfg properties.Jwt, now func() time.Time) []jwt.ParserOption {
	if now == nil {
		now = time.Now
	}
	return []jwt.ParserOption{
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithAudience(cfg.Audience),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(time.Duration(cfg.LeewaySeconds) * time.Second),
		jwt.WithStrictDecoding(),
		jwt.WithTimeFunc(now),
	}
}
