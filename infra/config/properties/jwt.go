package properties

import (
	"log"
)

type JwtProp struct {
	Jwt Jwt `yaml:"jwt"`
}

type Jwt struct {
	SecretKey string `yaml:"secret-key"`
}

func NewJwtProp() JwtProp {
	return JwtProp{}
}

func (d *JwtProp) Validate() {
	if d.Jwt.SecretKey == "" {
		log.Fatal("authBase.Jwt.SecretKey is empty")
	}
}
