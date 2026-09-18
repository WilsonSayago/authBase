package domain

import (
	"bytes"
	"testing"
	"time"
)

func TestHashRefreshTokenDeterministic(t *testing.T) {
	t.Parallel()

	a := HashRefreshToken("refresh-token-value")
	b := HashRefreshToken("refresh-token-value")
	c := HashRefreshToken("other-token-value")

	if a != b {
		t.Fatal("identical tokens must produce identical hashes")
	}
	if a == c {
		t.Fatal("different tokens must produce different hashes")
	}
	if bytes.Equal(a[:], []byte("refresh-token-value")) {
		t.Fatal("hash must not equal plaintext token")
	}
}

func TestRefreshSessionValid(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	valid := RefreshSession{
		TokenID:   "jti-1",
		FamilyID:  "family-1",
		UserID:    "user-1",
		TokenHash: HashRefreshToken("token"),
		IssuedAt:  now,
		ExpiresAt: now.Add(time.Hour),
	}
	if !valid.Valid() {
		t.Fatal("expected valid session")
	}

	tests := []struct {
		name   string
		mutate func(*RefreshSession)
	}{
		{name: "missing token id", mutate: func(s *RefreshSession) { s.TokenID = "" }},
		{name: "missing family id", mutate: func(s *RefreshSession) { s.FamilyID = "" }},
		{name: "missing user id", mutate: func(s *RefreshSession) { s.UserID = "" }},
		{name: "zero issued at", mutate: func(s *RefreshSession) { s.IssuedAt = time.Time{} }},
		{name: "zero expires at", mutate: func(s *RefreshSession) { s.ExpiresAt = time.Time{} }},
		{name: "expires before issued", mutate: func(s *RefreshSession) { s.ExpiresAt = s.IssuedAt.Add(-time.Second) }},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			session := valid
			tc.mutate(&session)
			if session.Valid() {
				t.Fatal("expected invalid session")
			}
		})
	}
}
