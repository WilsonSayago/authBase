package properties

import (
	"log"
)

type JwtProp struct {
	Jwt Jwt `yaml:"jwt"`
}

type Jwt struct {
	SecretKey        string `yaml:"secret-key"`
	RefreshSecret    string `yaml:"refresh-secret"`
	ExpirationTime   int    `yaml:"expiration-time"`
	RefreshTokenTime int    `yaml:"refresh-token-time"`
}

func NewJwtProp() JwtProp {
	return JwtProp{}
}

func (d *JwtProp) Validate() {
	if d.Jwt.SecretKey == "" {
		log.Fatal("authBase.Jwt.SecretKey is empty")
	}
}
