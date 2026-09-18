package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/WilsonSayago/authBase/v3/core"
	"github.com/WilsonSayago/authBase/v3/core/domain"
	"github.com/WilsonSayago/authBase/v3/core/port"
)

// contextUserKey is the private storage key for the authenticated user.
// Consumers must use GetUserToken; do not rely on this string outside authBase.
const contextUserKey = "authBase.internal.user"

var (
	publicUnauthorized = map[string]string{"error": "unauthorized"}
	publicForbidden    = map[string]string{"error": "forbidden"}
)

type Authorization[T domain.IUserGeneric, C core.Context] struct {
	users  port.UserReader[T]
	tokens *TokenManager
}

// NewAuthorization constructs an authorization middleware backed by TokenManager.
func NewAuthorization[T domain.IUserGeneric, C core.Context](
	users port.UserReader[T],
	tokens *TokenManager,
) (*Authorization[T, C], error) {
	if users == nil {
		return nil, fmt.Errorf("authorization user reader is nil")
	}
	if tokens == nil {
		return nil, fmt.Errorf("authorization token manager is nil")
	}
	return &Authorization[T, C]{
		users:  users,
		tokens: tokens,
	}, nil
}

func (a *Authorization[T, C]) AuthorizeJWT() func(ctx C) {
	return func(ctx C) {
		tokenString, ok := parseBearerToken(ctx.GetHeader("Authorization"))
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, publicUnauthorized)
			return
		}

		claims, err := a.tokens.ParseAccess(tokenString)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, publicUnauthorized)
			return
		}

		user, err := a.users.FindByID(requestContext(ctx), claims.Subject)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, publicUnauthorized)
			return
		}
		if !user.GetActive() {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, publicUnauthorized)
			return
		}

		ctx.Set(contextUserKey, user)
		ctx.Next()
	}
}

func (a *Authorization[T, C]) PoliciesGuard(
	fn func(C),
	fnValidate func(T, string, domain.OperationEnum) bool,
	entity string,
	operation domain.OperationEnum,
) func(C) {
	return func(c C) {
		user, ok := a.GetUserToken(c)
		if !ok || !user.GetActive() || user.GetId() == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, publicUnauthorized)
			return
		}
		if !a.IsAuthorized(user, fnValidate, entity, operation) {
			c.AbortWithStatusJSON(http.StatusForbidden, publicForbidden)
			return
		}
		fn(c)
	}
}

// GetUserToken returns the authenticated user previously stored by AuthorizeJWT.
func (a *Authorization[T, C]) GetUserToken(c core.Context) (T, bool) {
	var zero T
	value, exist := c.Get(contextUserKey)
	if !exist || value == nil {
		return zero, false
	}
	user, ok := value.(T)
	if !ok {
		return zero, false
	}
	return user, true
}

func (a *Authorization[T, C]) IsAuthorized(
	user T,
	fnValidate func(T, string, domain.OperationEnum) bool,
	entity string,
	operation domain.OperationEnum,
) bool {
	if user.GetId() == "" || !user.GetActive() {
		return false
	}
	if fnValidate != nil {
		return fnValidate(user, entity, operation)
	}
	if user.GetIsAdmin() {
		return true
	}
	return user.HasPermission(entity, operation)
}

func requestContext(ctx core.Context) context.Context {
	if reqCtx := ctx.RequestContext(); reqCtx != nil {
		return reqCtx
	}
	return context.Background()
}

// parseBearerToken accepts exactly "Bearer <token>" (scheme case-insensitive),
// with one token segment and no extra fields.
func parseBearerToken(header string) (string, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", false
	}
	parts := strings.Fields(header)
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	if parts[1] == "" {
		return "", false
	}
	return parts[1], true
}
