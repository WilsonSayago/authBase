package ports

type ValidationPort interface {
	HashPassword(password string) (string, error)
	CheckPassword(hashedPassword, password string) bool
}
