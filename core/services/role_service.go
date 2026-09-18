package services

import (
	"github.com/WilsonSayago/authBase/v3/core"
	"github.com/WilsonSayago/authBase/v3/core/domain"
	"github.com/WilsonSayago/authBase/v3/core/port"
)

type RoleService struct {
	port port.RolePort
}

func GetRoleServiceInstance(port port.RolePort) core.RoleUseCase {
	return &RoleService{
		port: port,
	}
}

func (r *RoleService) GetRoleById(id string) (domain.Role, error) {
	return r.port.FindById(id)
}

func (r *RoleService) GetRoles(pageSize, offset int) ([]domain.Role, int, error) {
	return r.port.FindAll(pageSize, offset)
}

func (r *RoleService) CreateRole(role domain.Role) (domain.Role, error) {
	return r.port.Save(role)
}

func (r *RoleService) UpdateRole(role domain.Role) error {
	return r.port.Update(role)
}

func (r *RoleService) ChangeStatus(id string) error {
	return r.port.ChangeStatus(id)
}
