package domain

// RoutePathEnum represents the enumeration of route paths.
// @Description RoutePathEnum
type RoutePathEnum string

const (
	USERS RoutePathEnum = "USERS"
	ROLES RoutePathEnum = "ROLES"
)

type Role struct {
	Base
	Name        string
	Routes      []RoutePathEnum
	Permissions []Permission
}

func (r *Role) GetName() string {
	return r.Name
}

func (r *Role) GetRoutes() []RoutePathEnum {
	return r.Routes
}

func (r *Role) GetPermissions() []Permission {
	return r.Permissions
}

func (r *Role) GetId() string {
	return r.Id
}

func (r *Role) GetActive() bool {
	return r.Active
}
