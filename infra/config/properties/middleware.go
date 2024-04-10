package properties

import (
	"log"
)

type MiddlewareProp struct {
	Middleware Middleware `yaml:"middleware"`
}

type Middleware struct {
	Jwt Jwt `yaml:"jwt"`
}

type Jwt struct {
	SecretKey string `yaml:"secret-key"`
}

func NewMiddlewareProp() MiddlewareProp {
	return MiddlewareProp{}
}

func (d *MiddlewareProp) Validate() {
	if d.Middleware.Jwt.SecretKey == "" {
		log.Fatal("Middleware.Jwt.SecretKey is empty")
	}
}
