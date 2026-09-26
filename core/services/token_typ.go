package services

// Distinct JWT typ values for access and refresh. Legacy HMAC tokens may use "JWT".
const (
	TokenTypAccess  = "at+jwt"
	TokenTypRefresh = "rt+jwt"
	tokenTypLegacy  = "JWT"
)

func typFor(tokenType TokenType) string {
	if tokenType == TokenTypeRefresh {
		return TokenTypRefresh
	}
	return TokenTypAccess
}
