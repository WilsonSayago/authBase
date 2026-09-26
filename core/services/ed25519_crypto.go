package services

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/WilsonSayago/authBase/v4/core/port"
	"github.com/golang-jwt/jwt/v5"
)

// Ed25519Key is a locally configured EdDSA key pair or public verification key.
type Ed25519Key struct {
	ID         string
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

func (k Ed25519Key) public() (ed25519.PublicKey, error) {
	if len(k.PublicKey) == ed25519.PublicKeySize {
		return k.PublicKey, nil
	}
	if len(k.PrivateKey) == ed25519.PrivateKeySize {
		pub, ok := k.PrivateKey.Public().(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("ed25519 public key is invalid")
		}
		return pub, nil
	}
	return nil, fmt.Errorf("ed25519 public key is required")
}

type ed25519Signer struct {
	typ string
	key Ed25519Key
}

func newEd25519Signer(tokenType TokenType, key Ed25519Key) (*ed25519Signer, error) {
	if strings.TrimSpace(key.ID) == "" {
		return nil, fmt.Errorf("ed25519 kid is required")
	}
	if len(key.PrivateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("ed25519 private key is required")
	}
	return &ed25519Signer{typ: typFor(tokenType), key: key}, nil
}

func (s *ed25519Signer) Sign(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["typ"] = s.typ
	token.Header["kid"] = s.key.ID
	signed, err := token.SignedString(s.key.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

type ed25519Verifier struct {
	expectedTyp string
	keys        map[string]ed25519.PublicKey
}

func newEd25519Verifier(tokenType TokenType, active Ed25519Key, previous []Ed25519Key) (*ed25519Verifier, error) {
	keys := map[string]ed25519.PublicKey{}
	if err := addEd25519Key(keys, active); err != nil {
		return nil, err
	}
	for _, key := range previous {
		if err := addEd25519Key(keys, key); err != nil {
			return nil, err
		}
	}
	return &ed25519Verifier{expectedTyp: typFor(tokenType), keys: keys}, nil
}

func addEd25519Key(keys map[string]ed25519.PublicKey, key Ed25519Key) error {
	if strings.TrimSpace(key.ID) == "" {
		return fmt.Errorf("ed25519 kid is required")
	}
	if _, exists := keys[key.ID]; exists {
		return fmt.Errorf("duplicate jwt kid")
	}
	pub, err := key.public()
	if err != nil {
		return err
	}
	keys[key.ID] = pub
	return nil
}

func (v *ed25519Verifier) Parse(tokenString string, claims jwt.Claims, opts ...jwt.ParserOption) (*jwt.Token, error) {
	return parseSignedToken(tokenString, claims, []string{jwt.SigningMethodEdDSA.Alg()}, v.keyFunc, v.expectedTyp, false, opts...)
}

func (v *ed25519Verifier) keyFunc(token *jwt.Token) (any, error) {
	if token.Method == nil || token.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
		return nil, fmt.Errorf("unexpected jwt algorithm")
	}
	if _, ok := lookupRemoteKeyLocator(token.Header); ok {
		return nil, fmt.Errorf("remote jwt key locators are not allowed")
	}
	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		return nil, fmt.Errorf("jwt kid is required")
	}
	key, ok := v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("unknown jwt kid")
	}
	return key, nil
}

// AccessPublicJWKS returns a JWKS document with access verification public keys only.
func AccessPublicJWKS(active Ed25519Key, previous ...Ed25519Key) ([]byte, error) {
	keys := make([]map[string]string, 0, 1+len(previous))
	all := append([]Ed25519Key{active}, previous...)
	seen := map[string]struct{}{}
	for _, key := range all {
		if _, dup := seen[key.ID]; dup {
			return nil, fmt.Errorf("duplicate jwt kid")
		}
		seen[key.ID] = struct{}{}
		pub, err := key.public()
		if err != nil {
			return nil, err
		}
		keys = append(keys, map[string]string{
			"kty": "OKP",
			"crv": "Ed25519",
			"x":   base64.RawURLEncoding.EncodeToString(pub),
			"kid": key.ID,
			"use": "sig",
			"alg": "EdDSA",
		})
	}
	return json.Marshal(map[string]any{"keys": keys})
}

var (
	_ port.TokenSigner   = (*ed25519Signer)(nil)
	_ port.TokenVerifier = (*ed25519Verifier)(nil)
)
