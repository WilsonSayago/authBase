package port

type GenericPort[T any] interface {
	FindByEmail(email string) (T, error)
	FindFullById(id string) (T, error)
}
