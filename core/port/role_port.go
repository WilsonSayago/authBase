package port

import (
	"context"

	"github.com/WilsonSayago/authBase/v4/core/domain"
)

type RolePort interface {
	FindById(ctx context.Context, id string) (domain.Role, error)
	FindAll(ctx context.Context, request domain.PageRequest) (domain.Page[domain.Role], error)
	Save(ctx context.Context, role domain.Role) (domain.Role, error)
	Update(ctx context.Context, role domain.Role) error
	SetActive(ctx context.Context, id string, active bool) error
}
