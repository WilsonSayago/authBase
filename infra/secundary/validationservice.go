package secundary

import (
	"github.com/WilsonSayago/authBase/v4/core/port"
	"golang.org/x/crypto/bcrypt"
)

type ValidationService struct{}

func NewValidationService() port.ValidationPort {
	return &ValidationService{}
}

func (v *ValidationService) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (v *ValidationService) CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
