package core

import (
	"context"

	"github.com/WilsonSayago/authBase/v4/core/domain"
)

type RoleUseCase interface {
	GetRoleById(ctx context.Context, id string) (domain.Role, error)
	GetRoles(ctx context.Context, request domain.PageRequest) (domain.Page[domain.Role], error)
	CreateRole(ctx context.Context, role domain.Role) (domain.Role, error)
	UpdateRole(ctx context.Context, role domain.Role) error
	SetActive(ctx context.Context, id string, active bool) error
}
