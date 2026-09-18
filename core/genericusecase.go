package core

import "context"

type GenericUseCase[T any] interface {
	FindByID(ctx context.Context, id string) (T, error)
}
