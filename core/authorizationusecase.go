package core

import (
	"context"

	domain "github.com/WilsonSayago/authBase/v3/core/domain"
)

// Context is a framework-neutral HTTP request adapter used by authorization middleware.
type Context interface {
	GetHeader(key string) string
	Set(key string, value interface{})
	AbortWithStatusJSON(code int, jsonObj interface{})
	Next()
	Get(key string) (value any, exists bool)
	Status(code int)
	// RequestContext returns the request-scoped context for cancellation and deadlines.
	RequestContext() context.Context
}

type AuthorizationUseCase[T any, C Context] interface {
	AuthorizeJWT() func(ctx C)
	PoliciesGuard(
		fn func(C),
		fnValidate func(T, string, domain.OperationEnum) bool,
		entity string,
		operation domain.OperationEnum,
	) func(C)
	GetUserToken(c Context) (T, bool)
	IsAuthorized(
		user T,
		fnValidate func(T, string, domain.OperationEnum) bool,
		entity string,
		operation domain.OperationEnum,
	) bool
}
