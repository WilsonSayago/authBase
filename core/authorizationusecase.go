package core

import (
	"context"

	domain "github.com/WilsonSayago/authBase/core/domain"
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
	PoliciesGuard(fn func(C),
		fnValidate func(interface{}, string, domain.OperationEnum) bool,
		entity string,
		operation domain.OperationEnum) func(C)
	GetUserToken(c Context) T
	IsAuthorized(
		user T,
		fnValidate func(interface{}, string, domain.OperationEnum) bool,
		entity string, operation domain.OperationEnum) bool
}
