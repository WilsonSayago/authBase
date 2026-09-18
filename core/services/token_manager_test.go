package services

import (
	"strings"
	"testing"
	"time"

	"github.com/WilsonSayago/authBase/v3/infra/config/properties"
	"github.com/golang-jwt/jwt/v5"
)

func validTokenCfg() properties.Jwt {
	return properties.Jwt{
		SecretKey:        strings.Repeat("A", properties.MinSecretBytes),
		RefreshSecret:    strings.Repeat("B", properties.MinSecretBytes),
		ExpirationTime:   1,
		RefreshTokenTime: 24,
		Issuer:           "authbase",
		Audience:         "authbase-api",
		LeewaySeconds:    0,
	}
}

func TestTokenManagerIssueAndParseRoundTrip(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	ids := []string{"family-1", "jti-access-1", "jti-refresh-1"}
	idx := 0
	tm, err := newTokenManagerForTest(validTokenCfg(), func() time.Time { return fixed }, func() (string, error) {
		id := ids[idx]
		idx++
		return id, nil
	})
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	access, refresh := mustIssuePair(t, tm, "user-1")
	if access == "" || refresh == "" {
		t.Fatal("mustIssuePair() returned empty tokens")
	}

	accessClaims, err := tm.ParseAccess(access)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}
	if accessClaims.Subject != "user-1" || accessClaims.TokenType != TokenTypeAccess || accessClaims.ID != "jti-access-1" {
		t.Fatalf("unexpected access claims: %+v", accessClaims)
	}
	if accessClaims.FamilyID != "" {
		t.Fatal("access token must not include family id")
	}

	refreshClaims, err := tm.ParseRefresh(refresh)
	if err != nil {
		t.Fatalf("ParseRefresh() error = %v", err)
	}
	if refreshClaims.Subject != "user-1" || refreshClaims.TokenType != TokenTypeRefresh || refreshClaims.ID != "jti-refresh-1" {
		t.Fatalf("unexpected refresh claims: %+v", refreshClaims)
	}
	if refreshClaims.FamilyID != "family-1" {
		t.Fatalf("refresh family = %q, want family-1", refreshClaims.FamilyID)
	}
}

func TestTokenManagerRejectsCrossType(t *testing.T) {
	t.Parallel()

	tm, err := NewTokenManager(validTokenCfg())
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	access, refresh := mustIssuePair(t, tm, "user-1")
	if _, err := tm.ParseAccess(refresh); err == nil {
		t.Fatal("ParseAccess(refresh) error = nil, want error")
	}
	if _, err := tm.ParseRefresh(access); err == nil {
		t.Fatal("ParseRefresh(access) error = nil, want error")
	}
}

func TestTokenManagerRejectsWrongSecret(t *testing.T) {
	t.Parallel()

	tm, err := NewTokenManager(validTokenCfg())
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	access, _ := mustIssuePair(t, tm, "user-1")

	otherCfg := validTokenCfg()
	otherCfg.SecretKey = strings.Repeat("C", properties.MinSecretBytes)
	otherCfg.RefreshSecret = strings.Repeat("D", properties.MinSecretBytes)
	other, err := NewTokenManager(otherCfg)
	if err != nil {
		t.Fatalf("NewTokenManager(other) error = %v", err)
	}
	if _, err := other.ParseAccess(access); err == nil {
		t.Fatal("ParseAccess() error = nil for foreign secret")
	}
}

func TestTokenManagerRejectsWrongAlgorithm(t *testing.T) {
	t.Parallel()

	cfg := validTokenCfg()
	tm, err := NewTokenManager(cfg)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	now := time.Now()
	claims := TokenClaims{
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ID:        "jti-1",
			Issuer:    cfg.Issuer,
			Audience:  []string{cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signed, err := token.SignedString([]byte(cfg.SecretKey))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	if _, err := tm.ParseAccess(signed); err == nil {
		t.Fatal("ParseAccess() error = nil for HS512 token")
	}
}

func TestTokenManagerRejectsMissingExpiration(t *testing.T) {
	t.Parallel()

	cfg := validTokenCfg()
	tm, err := NewTokenManager(cfg)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	now := time.Now()
	claims := TokenClaims{
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "user-1",
			ID:       "jti-1",
			Issuer:   cfg.Issuer,
			Audience: []string{cfg.Audience},
			IssuedAt: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.SecretKey))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	if _, err := tm.ParseAccess(signed); err == nil {
		t.Fatal("ParseAccess() error = nil for token without exp")
	}
}

func TestTokenManagerRejectsMissingIssuedAt(t *testing.T) {
	t.Parallel()

	cfg := validTokenCfg()
	tm, err := NewTokenManager(cfg)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	now := time.Now()
	claims := TokenClaims{
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ID:        "jti-1",
			Issuer:    cfg.Issuer,
			Audience:  []string{cfg.Audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			// IssuedAt intentionally omitted: WithIssuedAt alone does not require it.
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.SecretKey))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	if _, err := tm.ParseAccess(signed); err == nil {
		t.Fatal("ParseAccess() error = nil for token without iat")
	}
}

func TestTokenManagerRejectsExpired(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	tm, err := newTokenManagerForTest(validTokenCfg(), func() time.Time { return fixed }, nil)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	access, _ := mustIssuePair(t, tm, "user-1")

	tm.now = func() time.Time { return fixed.Add(2 * time.Hour) }
	if _, err := tm.ParseAccess(access); err == nil {
		t.Fatal("ParseAccess() error = nil for expired token")
	}
}

func TestTokenManagerRejectsWrongIssuerAudienceAndType(t *testing.T) {
	t.Parallel()

	cfg := validTokenCfg()
	tm, err := NewTokenManager(cfg)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	now := time.Now()

	tests := []struct {
		name   string
		mutate func(*TokenClaims)
	}{
		{name: "wrong issuer", mutate: func(c *TokenClaims) { c.Issuer = "other" }},
		{name: "wrong audience", mutate: func(c *TokenClaims) { c.Audience = []string{"other"} }},
		{name: "wrong type", mutate: func(c *TokenClaims) { c.TokenType = TokenTypeRefresh }},
		{name: "empty subject", mutate: func(c *TokenClaims) { c.Subject = "" }},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			claims := TokenClaims{
				TokenType: TokenTypeAccess,
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "user-1",
					ID:        "jti-1",
					Issuer:    cfg.Issuer,
					Audience:  []string{cfg.Audience},
					IssuedAt:  jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
				},
			}
			tc.mutate(&claims)
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			signed, err := token.SignedString([]byte(cfg.SecretKey))
			if err != nil {
				t.Fatalf("SignedString() error = %v", err)
			}
			if _, err := tm.ParseAccess(signed); err == nil {
				t.Fatal("ParseAccess() error = nil, want rejection")
			}
		})
	}
}

func TestTokenManagerRejectsEmptySubjectOnIssue(t *testing.T) {
	t.Parallel()

	tm, err := NewTokenManager(validTokenCfg())
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	if _, err := tm.issueInitialPair(""); err == nil {
		t.Fatal(`issueInitialPair("") error = nil, want error`)
	}
}

func TestTokenFamilyRotatedPairsShareFamily(t *testing.T) {
	t.Parallel()

	tm, err := NewTokenManager(validTokenCfg())
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	first, err := tm.issueInitialPair("user-1")
	if err != nil {
		t.Fatalf("issueInitialPair() error = %v", err)
	}
	second, err := tm.issueRotatedPair("user-1", first.Session.FamilyID)
	if err != nil {
		t.Fatalf("issueRotatedPair() error = %v", err)
	}
	if first.Session.FamilyID == "" || first.Session.FamilyID != second.Session.FamilyID {
		t.Fatalf("family ids = %q / %q", first.Session.FamilyID, second.Session.FamilyID)
	}
	if first.Session.TokenID == second.Session.TokenID {
		t.Fatal("rotated pair must use a new jti")
	}
	if first.Session.TokenHash == second.Session.TokenHash {
		t.Fatal("rotated pair must use a new hash")
	}

	accessClaims, err := tm.ParseAccess(first.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}
	if accessClaims.FamilyID != "" {
		t.Fatal("access token unexpectedly included family id")
	}
}


func TestNewTokenManagerRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	cfg := validTokenCfg()
	cfg.Issuer = ""
	if _, err := NewTokenManager(cfg); err == nil {
		t.Fatal("NewTokenManager() error = nil for invalid config")
	}
}

func FuzzParseTokenNeverPanics(f *testing.F) {
	tm, err := NewTokenManager(validTokenCfg())
	if err != nil {
		f.Fatal(err)
	}
	access, refresh := mustIssuePair(f, tm, "fuzz-user")
	f.Add(access)
	f.Add(refresh)
	f.Add("")
	f.Add("not-a-jwt")
	f.Add("eyJhbGciOiJub25lIn0.eyJzdWIiOiIxIn0.")

	f.Fuzz(func(t *testing.T, token string) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatalf("ParseAccess panicked: %v", rec)
			}
		}()
		_, _ = tm.ParseAccess(token)
		_, _ = tm.ParseRefresh(token)
	})
}
