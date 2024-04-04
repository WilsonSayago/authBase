package properties

import (
	"log"
	"sync"
)

var (
	middlewareOnce sync.Once
	middlewareProp *MiddlewareProp
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

func GetMiddlewareProp() *MiddlewareProp {
	middlewareOnce.Do(func() {
		middlewareProp = &MiddlewareProp{}
	})
	return middlewareProp
}

func (d *MiddlewareProp) Validate() {
	if d.Middleware.Jwt.SecretKey == "" {
		log.Fatal("Middleware.Jwt.SecretKey is empty")
	}
}
