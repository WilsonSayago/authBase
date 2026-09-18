package secundary

import (
	"strings"
	"testing"
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

func TestValidationHashPasswordRejectsTooLongPassword(t *testing.T) {
	t.Parallel()

	svc := &ValidationService{}
	tooLong := strings.Repeat("a", 73)

	hashed, err := svc.HashPassword(tooLong)
	if err == nil {
		t.Fatal("HashPassword() error = nil, want error for password longer than bcrypt limit")
	}
	if hashed != "" {
		t.Fatal("HashPassword() returned hash on error")
	}
}
