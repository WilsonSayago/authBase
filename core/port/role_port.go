package port

import (
	"context"

	"github.com/WilsonSayago/authBase/v3/core/domain"
)

type RolePort interface {
	FindById(ctx context.Context, id string) (domain.Role, error)
	FindAll(ctx context.Context, pageSize, offset int) ([]domain.Role, int, error)
	Save(ctx context.Context, role domain.Role) (domain.Role, error)
	Update(ctx context.Context, role domain.Role) error
	SetActive(ctx context.Context, id string, active bool) error
}
