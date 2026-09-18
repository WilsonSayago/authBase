package properties

import (
	"strings"
	"testing"
)

func validJwt() Jwt {
	return Jwt{
		SecretKey:        strings.Repeat("a", MinSecretBytes),
		RefreshSecret:    strings.Repeat("b", MinSecretBytes),
		ExpirationTime:   1,
		RefreshTokenTime: 24,
		Issuer:           "authbase-test",
		Audience:         "authbase-clients",
		LeewaySeconds:    30,
	}
}

func TestJwtPropValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Jwt)
		nilProp bool
		wantErr string
	}{
		{
			name:   "valid",
			mutate: func(*Jwt) {},
		},
		{
			name:    "nil prop",
			nilProp: true,
			wantErr: "nil",
		},
		{
			name:    "empty access secret",
			mutate:  func(j *Jwt) { j.SecretKey = "" },
			wantErr: "secret-key",
		},
		{
			name:    "short access secret",
			mutate:  func(j *Jwt) { j.SecretKey = strings.Repeat("a", MinSecretBytes-1) },
			wantErr: "secret-key",
		},
		{
			name:    "short refresh secret",
			mutate:  func(j *Jwt) { j.RefreshSecret = strings.Repeat("b", MinSecretBytes-1) },
			wantErr: "refresh-secret",
		},
		{
			name:    "identical secrets",
			mutate:  func(j *Jwt) { j.RefreshSecret = j.SecretKey },
			wantErr: "different",
		},
		{
			name:    "non-positive access expiration",
			mutate:  func(j *Jwt) { j.ExpirationTime = 0 },
			wantErr: "expiration-time",
		},
		{
			name:    "non-positive refresh expiration",
			mutate:  func(j *Jwt) { j.RefreshTokenTime = 0 },
			wantErr: "refresh-token-time",
		},
		{
			name:    "refresh not greater than access",
			mutate:  func(j *Jwt) { j.RefreshTokenTime = j.ExpirationTime },
			wantErr: "greater than expiration-time",
		},
		{
			name:    "empty issuer",
			mutate:  func(j *Jwt) { j.Issuer = "" },
			wantErr: "issuer",
		},
		{
			name:    "empty audience",
			mutate:  func(j *Jwt) { j.Audience = "" },
			wantErr: "audience",
		},
		{
			name:    "negative leeway",
			mutate:  func(j *Jwt) { j.LeewaySeconds = -1 },
			wantErr: "leeway-seconds",
		},
		{
			name:    "leeway above max",
			mutate:  func(j *Jwt) { j.LeewaySeconds = MaxLeewaySeconds + 1 },
			wantErr: "leeway-seconds",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.nilProp {
				var prop *JwtProp
				err := prop.Validate()
				if err == nil {
					t.Fatal("Validate() error = nil, want error")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Validate() error = %v, want substring %q", err, tc.wantErr)
				}
				return
			}

			cfg := validJwt()
			tc.mutate(&cfg)
			prop := &JwtProp{Jwt: cfg}
			err := prop.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}
