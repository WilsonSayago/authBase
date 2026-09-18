package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/WilsonSayago/authBase/core"
	"github.com/WilsonSayago/authBase/core/domain"
	"github.com/WilsonSayago/authBase/core/port"
	"github.com/WilsonSayago/authBase/infra/config/properties"
)

type Authorization[T domain.IUserGeneric, C core.Context] struct {
	users  port.UserReader[T]
	tokens *TokenManager
}

// NewAuthorization constructs an authorization middleware backed by TokenManager.
func NewAuthorization[T domain.IUserGeneric, C core.Context](
	users port.UserReader[T],
	cfg properties.Jwt,
) (*Authorization[T, C], error) {
	if users == nil {
		return nil, fmt.Errorf("authorization user reader is nil")
	}
	tokens, err := NewTokenManager(cfg)
	if err != nil {
		return nil, err
	}
	return &Authorization[T, C]{
		users:  users,
		tokens: tokens,
	}, nil
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
		claims, err := a.tokens.ParseAccess(tokenString)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}

		user, err := a.users.FindByID(requestContext(ctx), claims.Subject)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if !user.GetActive() {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		ctx.Set("user", user)
		ctx.Next()
	}
}

func requestContext(ctx core.Context) context.Context {
	if reqCtx := ctx.RequestContext(); reqCtx != nil {
		return reqCtx
	}
	return context.Background()
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

	if !user.GetActive() {
		return false
	}
	if fnValidate != nil {
		return fnValidate(user, entity, operation)
	}
	if user.GetId() == "" || (!user.GetIsAdmin() && !user.HasPermission(entity, operation)) {
		return false
	}
	return true
}
