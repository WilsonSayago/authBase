package properties

import (
	"fmt"
)

// MaxLeewaySeconds is the maximum accepted clock skew for JWT validation.
const MaxLeewaySeconds = 120

// MinSecretBytes is the minimum length required for access and refresh secrets.
const MinSecretBytes = 32

type JwtProp struct {
	Jwt Jwt `yaml:"jwt"`
}

type Jwt struct {
	SecretKey        string `yaml:"secret-key"`
	RefreshSecret    string `yaml:"refresh-secret"`
	ExpirationTime   int    `yaml:"expiration-time"`    // access token lifetime in hours
	RefreshTokenTime int    `yaml:"refresh-token-time"` // refresh token lifetime in hours; must be greater than ExpirationTime
	Issuer           string `yaml:"issuer"`
	Audience         string `yaml:"audience"`
	LeewaySeconds    int    `yaml:"leeway-seconds"` // clock skew tolerance; 0..MaxLeewaySeconds
}

func NewJwtProp() JwtProp {
	return JwtProp{}
}

// Validate checks JWT configuration and returns an error instead of terminating the process.
func (d *JwtProp) Validate() error {
	if d == nil {
		return fmt.Errorf("jwt configuration is nil")
	}
	return d.Jwt.Validate()
}

// Validate checks JWT configuration fields.
func (j Jwt) Validate() error {
	if len(j.SecretKey) < MinSecretBytes {
		return fmt.Errorf("jwt secret-key must be at least %d bytes", MinSecretBytes)
	}
	if len(j.RefreshSecret) < MinSecretBytes {
		return fmt.Errorf("jwt refresh-secret must be at least %d bytes", MinSecretBytes)
	}
	if j.SecretKey == j.RefreshSecret {
		return fmt.Errorf("jwt secret-key and refresh-secret must be different")
	}
	if j.ExpirationTime <= 0 {
		return fmt.Errorf("jwt expiration-time must be positive")
	}
	if j.RefreshTokenTime <= 0 {
		return fmt.Errorf("jwt refresh-token-time must be positive")
	}
	if j.RefreshTokenTime <= j.ExpirationTime {
		return fmt.Errorf("jwt refresh-token-time must be greater than expiration-time")
	}
	if j.Issuer == "" {
		return fmt.Errorf("jwt issuer must not be empty")
	}
	if j.Audience == "" {
		return fmt.Errorf("jwt audience must not be empty")
	}
	if j.LeewaySeconds < 0 {
		return fmt.Errorf("jwt leeway-seconds must not be negative")
	}
	if j.LeewaySeconds > MaxLeewaySeconds {
		return fmt.Errorf("jwt leeway-seconds must be <= %d", MaxLeewaySeconds)
	}
	return nil
}
