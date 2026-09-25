package secundary

import (
	"strings"

	"github.com/WilsonSayago/authBase/v4/core/port"
	"golang.org/x/crypto/bcrypt"
)

type ValidationService struct{}

func NewValidationService() port.ValidationPort {
	return &ValidationService{}
}

func (v *ValidationService) HashPassword(password string) (string, error) {
	return hashArgon2id(password)
}

func (v *ValidationService) CheckPassword(hashedPassword, password string) bool {
	switch {
	case strings.HasPrefix(hashedPassword, "$argon2id$"):
		return verifyArgon2id(hashedPassword, password)
	case strings.HasPrefix(hashedPassword, "$2a$"),
		strings.HasPrefix(hashedPassword, "$2b$"),
		strings.HasPrefix(hashedPassword, "$2y$"):
		if len(password) > 72 {
			return false
		}
		return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
	default:
		return false
	}
}
