package domain

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

// generate getters for Role struct
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
