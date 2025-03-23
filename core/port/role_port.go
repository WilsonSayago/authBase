package port

import "github.com/WilsonSayago/authBase/core/domain"

type RolePort interface {
	FindById(id string) (domain.Role, error)
	FindAll(pageSize, offset int) ([]domain.Role, int, error)
	Save(role domain.Role) (domain.Role, error)
	Update(role domain.Role) error
	ChangeStatus(id string) error
}
