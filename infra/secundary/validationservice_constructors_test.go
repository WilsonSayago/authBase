package secundary

import (
	"sync"
	"testing"
)

func TestNewValidationServiceConstructor(t *testing.T) {
	t.Parallel()

	first := NewValidationService()
	second := NewValidationService()

	if first == nil || second == nil {
		t.Fatal("NewValidationService() returned nil")
	}
	if _, ok := first.(*ValidationService); !ok {
		t.Fatalf("first type = %T, want *ValidationService", first)
	}
	if _, ok := second.(*ValidationService); !ok {
		t.Fatalf("second type = %T, want *ValidationService", second)
	}

	// Empty structs may share addresses in Go; independence is verified by
	// successful concurrent construction and by hashing through each value.
	hashA, err := first.HashPassword("alpha-password")
	if err != nil {
		t.Fatalf("first.HashPassword() error = %v", err)
	}
	hashB, err := second.HashPassword("beta-password")
	if err != nil {
		t.Fatalf("second.HashPassword() error = %v", err)
	}
	if !first.CheckPassword(hashA, "alpha-password") {
		t.Fatal("first instance failed its own round-trip")
	}
	if !second.CheckPassword(hashB, "beta-password") {
		t.Fatal("second instance failed its own round-trip")
	}
	if first.CheckPassword(hashA, "beta-password") {
		t.Fatal("first instance accepted the other password")
	}
}

func TestNewValidationServiceConcurrentIndependentInstances(t *testing.T) {
	t.Parallel()

	const n = 100
	results := make([]portValidation, n)
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			got := NewValidationService()
			if got == nil {
				t.Errorf("instance %d is nil", i)
				return
			}
			if _, ok := got.(*ValidationService); !ok {
				t.Errorf("instance %d type = %T", i, got)
				return
			}
			results[i] = got
		}()
	}
	wg.Wait()

	for i, svc := range results {
		if svc == nil {
			t.Fatalf("missing instance at index %d", i)
		}
	}
}

// Avoid importing port in assertions while keeping a local alias for clarity.
type portValidation interface {
	HashPassword(password string) (string, error)
	CheckPassword(hashedPassword, password string) bool
}
