package ports

type GenericPort[T any] interface {
	FindById(id string) (T, error)
}
