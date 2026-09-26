package services

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHMACLegacyFixtureWithoutKidStillVerifies(t *testing.T) {
	t.Parallel()
	cfg := validTokenCfg()
	tm, err := NewTokenManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	claims := TokenClaims{
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ID:        "jti-legacy",
			Issuer:    cfg.Issuer,
			Audience:  []string{cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.SecretKey))
	if err != nil {
		t.Fatal(err)
	}
	got, err := tm.ParseAccess(signed)
	if err != nil {
		t.Fatalf("legacy HMAC without kid: %v", err)
	}
	if got.Subject != "user-1" || got.ID != "jti-legacy" {
		t.Fatalf("claims=%+v", got)
	}
}

func TestHMACCryptoWithKeysSignsKidAndVerifiesLegacy(t *testing.T) {
	t.Parallel()
	cfg := validTokenCfg()
	crypto, err := NewHMACCryptoWithKeys(
		HMACKey{ID: "access-a", Secret: []byte(cfg.SecretKey)},
		HMACKey{ID: "refresh-a", Secret: []byte(cfg.RefreshSecret)},
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	tm, err := NewTokenManagerWithCrypto(cfg, crypto)
	if err != nil {
		t.Fatal(err)
	}
	access, _ := mustIssuePair(t, tm, "user-1")
	token, _, err := jwt.NewParser().ParseUnverified(access, jwt.MapClaims{})
	if err != nil {
		t.Fatal(err)
	}
	if token.Header["kid"] != "access-a" {
		t.Fatalf("kid=%v", token.Header["kid"])
	}
	legacy := jwt.NewWithClaims(jwt.SigningMethodHS256, TokenClaims{
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user-1", ID: "legacy", Issuer: cfg.Issuer, Audience: []string{cfg.Audience},
			IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	signed, err := legacy.SignedString([]byte(cfg.SecretKey))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tm.ParseAccess(signed); err != nil {
		t.Fatalf("legacy without kid: %v", err)
	}
}

func TestNewHMACTokensSetAccessTyp(t *testing.T) {
	t.Parallel()
	tm, err := NewTokenManager(validTokenCfg())
	if err != nil {
		t.Fatal(err)
	}
	access, refresh := mustIssuePair(t, tm, "user-1")
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	accessTok, _, err := parser.ParseUnverified(access, &TokenClaims{})
	if err != nil {
		t.Fatal(err)
	}
	if accessTok.Header["typ"] != TokenTypAccess {
		t.Fatalf("access typ=%v", accessTok.Header["typ"])
	}
	if _, ok := accessTok.Header["kid"]; ok {
		t.Fatal("default HMAC tokens must omit kid")
	}
	refreshTok, _, err := parser.ParseUnverified(refresh, &TokenClaims{})
	if err != nil {
		t.Fatal(err)
	}
	if refreshTok.Header["typ"] != TokenTypRefresh {
		t.Fatalf("refresh typ=%v", refreshTok.Header["typ"])
	}
}

func TestEd25519RoundTripAndUnknownKid(t *testing.T) {
	t.Parallel()
	cfg := validTokenCfg()
	accessKey := mustEd25519(t, "access-1")
	refreshKey := mustEd25519(t, "refresh-1")
	crypto, err := NewEd25519Crypto(accessKey, refreshKey, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	tm, err := NewTokenManagerWithCrypto(cfg, crypto)
	if err != nil {
		t.Fatal(err)
	}
	access, refresh := mustIssuePair(t, tm, "user-1")
	if _, err := tm.ParseAccess(access); err != nil {
		t.Fatal(err)
	}
	if _, err := tm.ParseRefresh(refresh); err != nil {
		t.Fatal(err)
	}
	if _, err := tm.ParseAccess(refresh); err == nil {
		t.Fatal("cross type must fail")
	}

	other := mustEd25519(t, "access-other")
	foreign, err := NewEd25519Crypto(other, mustEd25519(t, "refresh-other"), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	otherTM, err := NewTokenManagerWithCrypto(cfg, foreign)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := otherTM.ParseAccess(access); err == nil {
		t.Fatal("unknown kid must fail")
	}
}

func TestEd25519RejectsAccessKeyOnRefresh(t *testing.T) {
	t.Parallel()
	accessKey := mustEd25519(t, "shared")
	crypto, err := NewEd25519Crypto(accessKey, accessKey, nil, nil)
	if err == nil {
		t.Fatalf("expected error, got %#v", crypto)
	}
}

func TestRotationOverlapThenRetire(t *testing.T) {
	t.Parallel()
	cfg := validTokenCfg()
	oldAccess, oldRefresh := mustEd25519(t, "access-old"), mustEd25519(t, "refresh-old")
	newAccess, newRefresh := mustEd25519(t, "access-new"), mustEd25519(t, "refresh-new")

	oldCrypto, err := NewEd25519Crypto(oldAccess, oldRefresh, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldTM, err := NewTokenManagerWithCrypto(cfg, oldCrypto)
	if err != nil {
		t.Fatal(err)
	}
	oldAccessTok, oldRefreshTok := mustIssuePair(t, oldTM, "user-1")

	overlapAccess, err := newEd25519Verifier(TokenTypeAccess, newAccess, []Ed25519Key{oldAccess})
	if err != nil {
		t.Fatal(err)
	}
	overlapRefresh, err := newEd25519Verifier(TokenTypeRefresh, newRefresh, []Ed25519Key{oldRefresh})
	if err != nil {
		t.Fatal(err)
	}
	newSigner, err := NewEd25519Crypto(newAccess, newRefresh, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	overlap, err := NewTokenManagerWithCrypto(cfg, TokenCrypto{
		AccessSigner:    newSigner.AccessSigner,
		RefreshSigner:   newSigner.RefreshSigner,
		AccessVerifier:  overlapAccess,
		RefreshVerifier: overlapRefresh,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := overlap.ParseAccess(oldAccessTok); err != nil {
		t.Fatalf("overlap must verify old access: %v", err)
	}
	if _, err := overlap.ParseRefresh(oldRefreshTok); err != nil {
		t.Fatalf("overlap must verify old refresh: %v", err)
	}
	newAccessTok, _ := mustIssuePair(t, overlap, "user-1")
	if _, err := overlap.ParseAccess(newAccessTok); err != nil {
		t.Fatal(err)
	}

	retired, err := NewTokenManagerWithCrypto(cfg, newSigner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := retired.ParseAccess(oldAccessTok); err == nil {
		t.Fatal("retired verifier must reject old kid")
	}
	if _, err := retired.ParseAccess(newAccessTok); err != nil {
		t.Fatal(err)
	}
}

func TestHMACAndEd25519CompositeOverlap(t *testing.T) {
	t.Parallel()
	cfg := validTokenCfg()
	hmacTM, err := NewTokenManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	hmacAccess, _ := mustIssuePair(t, hmacTM, "user-1")

	edCrypto, err := NewEd25519Crypto(mustEd25519(t, "access-1"), mustEd25519(t, "refresh-1"), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	hmacCrypto, err := NewHMACCrypto(cfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	accessV, err := NewCompositeVerifier(edCrypto.AccessVerifier, hmacCrypto.AccessVerifier)
	if err != nil {
		t.Fatal(err)
	}
	refreshV, err := NewCompositeVerifier(edCrypto.RefreshVerifier, hmacCrypto.RefreshVerifier)
	if err != nil {
		t.Fatal(err)
	}
	overlap, err := NewTokenManagerWithCrypto(cfg, TokenCrypto{
		AccessSigner:    edCrypto.AccessSigner,
		RefreshSigner:   edCrypto.RefreshSigner,
		AccessVerifier:  accessV,
		RefreshVerifier: refreshV,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := overlap.ParseAccess(hmacAccess); err != nil {
		t.Fatalf("composite must verify HMAC: %v", err)
	}
	edAccess, _ := mustIssuePair(t, overlap, "user-1")
	if _, err := overlap.ParseAccess(edAccess); err != nil {
		t.Fatal(err)
	}
	if _, err := hmacTM.ParseAccess(edAccess); err == nil {
		t.Fatal("HMAC-only manager must not verify EdDSA")
	}
}

func TestAccessPublicJWKSOmitsPrivateAndRefresh(t *testing.T) {
	t.Parallel()
	access := mustEd25519(t, "access-1")
	refresh := mustEd25519(t, "refresh-1")
	raw, err := AccessPublicJWKS(access)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if strings.Contains(body, "\"d\"") || strings.Contains(body, refresh.ID) {
		t.Fatalf("jwks leaked private/refresh material: %s", body)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	keys, _ := doc["keys"].([]any)
	if len(keys) != 1 {
		t.Fatalf("keys=%v", keys)
	}
	key, _ := keys[0].(map[string]any)
	if key["kid"] != "access-1" || key["kty"] != "OKP" || key["crv"] != "Ed25519" {
		t.Fatalf("key=%v", key)
	}
}

func TestRemoteKeyLocatorRejected(t *testing.T) {
	t.Parallel()
	cfg := validTokenCfg()
	tm, err := NewTokenManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	claims := TokenClaims{
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user-1", ID: "jti-1", Issuer: cfg.Issuer, Audience: []string{cfg.Audience},
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["jku"] = "https://evil.example/jwks"
	signed, err := token.SignedString([]byte(cfg.SecretKey))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tm.ParseAccess(signed); err == nil {
		t.Fatal("jku must be rejected")
	}
}

func mustEd25519(t *testing.T, kid string) Ed25519Key {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return Ed25519Key{ID: kid, PrivateKey: priv, PublicKey: pub}
}
