package core

import "github.com/WilsonSayago/authBase/core/domain"

type RoleUseCase interface {
	GetRoleById(id string) (domain.Role, error)
	GetRoles(pageSize, offset int) ([]domain.Role, int, error)
	CreateRole(role domain.Role) (domain.Role, error)
	UpdateRole(role domain.Role) error
	ChangeStatus(id string) error
}
