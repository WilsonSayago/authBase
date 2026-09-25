package secundary

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestValidationHashPasswordDiffersFromInput(t *testing.T) {
	t.Parallel()

	svc := &ValidationService{}
	password := "correct-horse-battery"

	hashed, err := svc.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hashed == "" {
		t.Fatal("HashPassword() returned empty hash")
	}
	if hashed == password {
		t.Fatal("HashPassword() returned plaintext password")
	}
	if !strings.HasPrefix(hashed, "$argon2id$v=19$") {
		t.Fatalf("HashPassword() = %q, want argon2id PHC", hashed)
	}
}

func TestValidationCheckPasswordRoundTrip(t *testing.T) {
	t.Parallel()

	svc := &ValidationService{}
	password := "round-trip-secret"

	hashed, err := svc.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !svc.CheckPassword(hashed, password) {
		t.Fatal("CheckPassword() = false for matching password")
	}
}

func TestValidationCheckPasswordRejectsWrongPassword(t *testing.T) {
	t.Parallel()

	svc := &ValidationService{}
	hashed, err := svc.HashPassword("expected-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if svc.CheckPassword(hashed, "wrong-password") {
		t.Fatal("CheckPassword() = true for wrong password")
	}
}

func TestValidationCheckPasswordRejectsInvalidHash(t *testing.T) {
	t.Parallel()

	svc := &ValidationService{}
	if svc.CheckPassword("not-a-bcrypt-hash", "any-password") {
		t.Fatal("CheckPassword() = true for invalid hash")
	}
}

func TestValidationHashPasswordUsesDistinctSalts(t *testing.T) {
	t.Parallel()

	svc := &ValidationService{}
	first, err := svc.HashPassword("same-password")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.HashPassword("same-password")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("HashPassword() reused an Argon2id salt")
	}
}

func TestValidationHashPasswordRejectsDefensiveByteLimit(t *testing.T) {
	t.Parallel()

	svc := &ValidationService{}
	tooLong := strings.Repeat("a", MaxPasswordBytes+1)

	hashed, err := svc.HashPassword(tooLong)
	if err == nil {
		t.Fatal("HashPassword() error = nil, want defensive length error")
	}
	if hashed != "" {
		t.Fatal("HashPassword() returned hash on error")
	}
}

func TestValidationCheckPasswordSupportsLegacyBcrypt(t *testing.T) {
	t.Parallel()

	legacy, err := bcrypt.GenerateFromPassword([]byte("legacy-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	svc := &ValidationService{}
	if !svc.CheckPassword(string(legacy), "legacy-password") {
		t.Fatal("CheckPassword() rejected legacy bcrypt")
	}
	if svc.CheckPassword(string(legacy), "wrong-password") {
		t.Fatal("CheckPassword() accepted wrong legacy bcrypt password")
	}
	if svc.CheckPassword(string(legacy), strings.Repeat("x", 73)) {
		t.Fatal("CheckPassword() accepted bcrypt password over 72 bytes")
	}
}

func TestValidationCheckPasswordRejectsMalformedArgon2id(t *testing.T) {
	t.Parallel()

	svc := &ValidationService{}
	tests := []string{
		"$argon2id$",
		"$argon2id$v=18$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=999999999,t=2,p=1$MDEyMzQ1Njc4OWFiY2RlZg$MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY",
		"$argon2id$v=19$m=19456,t=999,p=1$MDEyMzQ1Njc4OWFiY2RlZg$MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY",
		"$argon2id$v=19$m=19456,t=2,p=99$MDEyMzQ1Njc4OWFiY2RlZg$MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY",
		"$argon2id$v=19$m=19456,t=2,p=1$not-base64!$also-not-base64!",
		"$argon2id$" + strings.Repeat("A", 513),
	}
	for _, encoded := range tests {
		if svc.CheckPassword(encoded, "password") {
			t.Fatalf("CheckPassword() accepted malformed PHC %q", encoded)
		}
	}
}

func BenchmarkValidationServiceHashPassword(b *testing.B) {
	svc := &ValidationService{}
	for i := 0; i < b.N; i++ {
		if _, err := svc.HashPassword("benchmark-password"); err != nil {
			b.Fatal(err)
		}
	}
}
