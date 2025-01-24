package services

import (
	"fmt"
	"github.com/WilsonSayago/authBase/core"
	"github.com/WilsonSayago/authBase/core/domain"
	"github.com/WilsonSayago/authBase/core/port"
	"github.com/WilsonSayago/authBase/infra/config/properties"
	"github.com/WilsonSayago/initModules"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
)

type Authorization[T domain.IUserGeneric, C core.Context] struct {
	port port.GenericPort[T]
	prop *properties.JwtProp
}

func NewAuthorization[T domain.IUserGeneric, C core.Context](port port.GenericPort[T], prop *properties.JwtProp) core.AuthorizationUseCase[T, C] {
	instance := initModules.GetInstance("Authorization", func() interface{} {
		return &Authorization[T, C]{
			port: port,
			prop: prop,
		}
	})
	return instance.(*Authorization[T, C])
}

func (a *Authorization[T, C]) AuthorizeJWT() func(ctx C) {
	return func(ctx C) {
		const BearerSchema = "Bearer "
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{"error": "It's necessary authorization header"})
			return
		}

		tokenString := authHeader[len(BearerSchema):]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(a.prop.Jwt.SecretKey), nil
		})

		if token != nil && token.Valid {
			userId := token.Claims.(jwt.MapClaims)["user"].(string)
			user, err := a.port.FindFullById(userId)
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
				return
			}
			ctx.Set("user", user)
			ctx.Next()
		} else {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		}
	}
}

func (a *Authorization[T, C]) PoliciesGuard(fn func(C),
	fnValidate func(interface{}, string, domain.OperationEnum) bool,
	entity string,
	operation domain.OperationEnum) func(C) {
	return func(c C) {
		user := a.GetUserToken(c)
		if !a.IsAuthorized(user, fnValidate, entity, operation) {
			c.Status(http.StatusUnauthorized)
			return
		}
		fn(c)
	}
}

func (a *Authorization[T, C]) GetUserToken(c core.Context) T {
	user, exist := c.Get("user")
	if !exist {
		return interface{}(nil).(T)
	}
	return user.(T)
}

func (a *Authorization[T, C]) IsAuthorized(
	user T,
	fnValidate func(interface{}, string, domain.OperationEnum) bool,
	entity string, operation domain.OperationEnum) bool {

	if fnValidate != nil {
		return fnValidate(user, entity, operation)
	} else if user.GetId() == "" || (!user.GetIsAdmin() && !user.HasPermission(entity, operation)) {
		return false
	}
	return true
}
