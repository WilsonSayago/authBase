package core

type GenericUseCase[T any] interface {
	FindFullById(id string) (T, error)
}
