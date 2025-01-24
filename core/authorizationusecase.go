package core

import domain "github.com/WilsonSayago/authBase/core/domain"

type Context interface {
	GetHeader(key string) string
	Set(key string, value interface{})
	AbortWithStatusJSON(code int, jsonObj interface{})
	Next()
	Get(key string) (value any, exists bool)
	Status(code int)
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
