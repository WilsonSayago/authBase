package core

import "github.com/WilsonSayago/authBase/core/domain"

type RoleUseCase interface {
	GetRoleById(id string) (domain.Role, error)
	GetRoles() ([]domain.Role, error)
	CreateRole(role domain.Role) (domain.Role, error)
}
