package services

import (
	"fmt"
	"github.com/WilsonSayago/middleware/core"
	"github.com/WilsonSayago/middleware/core/domains"
	"github.com/WilsonSayago/middleware/core/ports"
	"github.com/WilsonSayago/middleware/infra/config/properties"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"sync"
	"time"
)

type MyCustomClaims struct {
	User string `json:"user"`
	jwt.RegisteredClaims
}

var (
	serviceSingleton core.MiddlewareUseCase
	serviceOnce      sync.Once
)

type Middleware[T any] struct {
	port ports.GenericPort[T]
	prop *properties.MiddlewareProp
}

func GetMiddlewareInstance[T any](port ports.GenericPort[T], prop *properties.MiddlewareProp) core.MiddlewareUseCase {
	serviceOnce.Do(func() {
		service := &Middleware[T]{
			port: port,
			prop: prop,
		}
		var IService core.MiddlewareUseCase = service
		serviceSingleton = IService
	})
	return serviceSingleton
}

func (m *Middleware[T]) AuthorizeJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		const BearerSchema = "Bearer "
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "It's necessary authorization header"})
			return
		}
		
		tokenString := authHeader[len(BearerSchema):]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return m.prop.Middleware.Jwt.SecretKey, nil
		})
		
		if token != nil && token.Valid {
			userId := token.Claims.(jwt.MapClaims)["user"].(string)
			user, err := m.port.FindById(userId)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}
			c.Set("user", user)
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		}
	}
}

func (m *Middleware[T]) GetToken(userId string) (string, error) {
	claims := MyCustomClaims{
		userId,
		jwt.RegisteredClaims{
			// A usual scenario is to set the expiration time relative to the current time
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			//IssuedAt:  jwt.NewNumericDate(time.Now()),
			//NotBefore: jwt.NewNumericDate(time.Now()),
			//Issuer:    "test",
			//Subject:   "somebody",
			//ID:        "1",
			//Audience:  []string{"somebody_else"},
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte(m.prop.Middleware.Jwt.SecretKey))
	
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return ss, nil
}

func PoliciesGuard(fn gin.HandlerFunc, entity domain.EntityEnum, operation domain.OperationEnum) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUserToken[domain.UserGeneric](c)
		if user.Id == "" || (!user.IsAdmin && !user.HasPermission(entity, operation)) {
			c.Status(http.StatusUnauthorized)
			return
		}
		fn(c)
	}
}

func GetUserToken[T any](c *gin.Context) T {
	user, exist := c.Get("user")
	if !exist {
		var zeroValue T
		return zeroValue
	}
	return user.(T)
}
