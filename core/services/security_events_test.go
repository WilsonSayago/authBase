package services

import (
	"context"
	"sync"
	"testing"

	"github.com/WilsonSayago/authBase/v4/core"
)

type memSink struct {
	mu     sync.Mutex
	events []core.SecurityEvent
	panic  bool
}

func (s *memSink) Record(_ context.Context, event core.SecurityEvent) {
	if s.panic {
		panic("sink boom")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *memSink) types() []core.SecurityEventType {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]core.SecurityEventType, 0, len(s.events))
	for _, event := range s.events {
		out = append(out, event.Type)
	}
	return out
}

func TestSecurityEventsLoginAndPanicIsolation(t *testing.T) {
	t.Parallel()
	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	validate := &fakeValidationPort{
		checkPassword: func(hashedPassword, password string) bool {
			return hashedPassword == "stored-hash" && password == "plain-password"
		},
	}
	sink := &memSink{}
	svc := newTestAuthService(t, store, validate).WithSecurityEvents(sink)

	if _, _, err := svc.Login(context.Background(), "ada@example.com", "plain-password"); err != nil {
		t.Fatal(err)
	}
	if got := sink.types(); len(got) != 1 || got[0] != core.SecurityLoginSuccess {
		t.Fatalf("events=%v", got)
	}

	boom := &memSink{panic: true}
	svc.WithSecurityEvents(boom)
	if _, _, err := svc.Login(context.Background(), "ada@example.com", "plain-password"); err != nil {
		t.Fatalf("sink panic must not fail login: %v", err)
	}
}

func TestSecurityEventsOmitSecrets(t *testing.T) {
	t.Parallel()
	store := newFakeIdentityStore()
	store.add(fakeUser{id: "user-1", email: "ada@example.com", active: true}, "stored-hash")
	validate := &fakeValidationPort{
		checkPassword: func(string, string) bool { return false },
	}
	sink := &memSink{}
	svc := newTestAuthService(t, store, validate).WithSecurityEvents(sink)
	_, _, err := svc.Login(context.Background(), "ada@example.com", "wrong-password")
	if err != core.ErrInvalidCredentials {
		t.Fatalf("err=%v", err)
	}
	if len(sink.events) != 1 {
		t.Fatalf("events=%d", len(sink.events))
	}
	event := sink.events[0]
	if event.Type != core.SecurityLoginFailure || event.ActorID != "" {
		t.Fatalf("event=%+v", event)
	}
}
