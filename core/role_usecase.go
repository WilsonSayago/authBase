package core

import (
	"context"

	"github.com/WilsonSayago/authBase/v3/core/domain"
)

type RoleUseCase interface {
	GetRoleById(ctx context.Context, id string) (domain.Role, error)
	GetRoles(ctx context.Context, pageSize, offset int) ([]domain.Role, int, error)
	CreateRole(ctx context.Context, role domain.Role) (domain.Role, error)
	UpdateRole(ctx context.Context, role domain.Role) error
	SetActive(ctx context.Context, id string, active bool) error
}
