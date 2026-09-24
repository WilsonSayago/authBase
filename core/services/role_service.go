package services

import (
	"context"

	"github.com/WilsonSayago/authBase/v4/core"
	"github.com/WilsonSayago/authBase/v4/core/domain"
	"github.com/WilsonSayago/authBase/v4/core/port"
)

type RoleService struct {
	port port.RolePort
}

func GetRoleServiceInstance(port port.RolePort) core.RoleUseCase {
	return &RoleService{
		port: port,
	}
}

func (r *RoleService) GetRoleById(ctx context.Context, id string) (domain.Role, error) {
	return r.port.FindById(ctx, id)
}

func (r *RoleService) GetRoles(ctx context.Context, request domain.PageRequest) (domain.Page[domain.Role], error) {
	return r.port.FindAll(ctx, request)
}

func (r *RoleService) CreateRole(ctx context.Context, role domain.Role) (domain.Role, error) {
	return r.port.Save(ctx, role)
}

func (r *RoleService) UpdateRole(ctx context.Context, role domain.Role) error {
	return r.port.Update(ctx, role)
}

func (r *RoleService) SetActive(ctx context.Context, id string, active bool) error {
	return r.port.SetActive(ctx, id, active)
}
