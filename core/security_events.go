package core

import "context"

// SecurityEventType is a closed set of authentication/session events.
type SecurityEventType string

const (
	SecurityLoginSuccess   SecurityEventType = "login_success"
	SecurityLoginFailure   SecurityEventType = "login_failure"
	SecurityRefreshSuccess SecurityEventType = "refresh_success"
	SecurityRefreshReplay  SecurityEventType = "refresh_replay"
	SecurityFamilyRevoke   SecurityEventType = "family_revoke"
	SecurityUserRevoke     SecurityEventType = "user_revoke"
	SecurityTokenReject    SecurityEventType = "token_reject"
	SecurityKeyRotation    SecurityEventType = "key_rotation"
	SecurityOIDCStart      SecurityEventType = "oidc_start"
	SecurityOIDCSuccess    SecurityEventType = "oidc_success"
	SecurityOIDCFailure    SecurityEventType = "oidc_failure"
	SecurityOIDCLink       SecurityEventType = "oidc_link"
	SecurityOIDCConflict   SecurityEventType = "oidc_conflict"
)

// SecurityEvent is an observability record without secrets or raw PII.
type SecurityEvent struct {
	Type     SecurityEventType
	ActorID  string
	FamilyID string
	TokenID  string
	Provider string
	Outcome  string
}

// SecurityEventSink receives typed events. It must not influence auth decisions.
type SecurityEventSink interface {
	Record(ctx context.Context, event SecurityEvent)
}

// NoopSecurityEventSink discards events.
type NoopSecurityEventSink struct{}

func (NoopSecurityEventSink) Record(context.Context, SecurityEvent) {}
