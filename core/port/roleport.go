package port

import "github.com/WilsonSayago/authBase/core/domain"

type RolePort interface {
	FindById(id string) (domain.Role, error)
	FindAll() ([]domain.Role, error)
	Save(role domain.Role) (domain.Role, error)
}
