package domain

type IUserGeneric interface {
	GetId() string
	GetEmail() string
	GetPassword() string
	GetPermissions() []Permission
	GetIsAdmin() bool
	HasPermission(entity string, operation OperationEnum) bool
}

type UserGeneric struct {
	Base
	name        string
	email       string
	password    string
	permissions []Permission
	isAdmin     bool
	roles       []Role
}

func NewUserGeneric(id, name, email, password string, roles []Role, isAdmin bool) UserGeneric {
	return UserGeneric{Base: Base{
		Id: id,
	}, name: name,
		email:    email,
		password: password,
		roles:    roles,
		isAdmin:  isAdmin,
	}
}

func (u UserGeneric) GetName() string {
	return u.name
}

func (u UserGeneric) GetId() string {
	return u.Id
}

func (u UserGeneric) GetEmail() string {
	return u.email
}

func (u UserGeneric) GetPassword() string {
	return u.password
}

func (u UserGeneric) GetPermissions() []Permission {
	permissionMap := make(map[string]Permission)
	for _, role := range u.roles {
		for _, perm := range role.Permissions {
			if existingPerm, exists := permissionMap[perm.Entity]; exists {
				permissionMap[perm.Entity] = Permission{
					Entity: perm.Entity,
					Create: existingPerm.Create || perm.Create,
					Read:   existingPerm.Read || perm.Read,
					Update: existingPerm.Update || perm.Update,
					Delete: existingPerm.Delete || perm.Delete,
				}
			} else {
				permissionMap[perm.Entity] = perm
			}
		}
	}
	uniquePermissions := make([]Permission, 0, len(permissionMap))
	for _, perm := range permissionMap {
		uniquePermissions = append(uniquePermissions, perm)
	}
	return uniquePermissions
}

func (u UserGeneric) GetIsAdmin() bool {
	return u.isAdmin
}

func (u UserGeneric) HasPermission(entity string, operation OperationEnum) bool {
	for _, permission := range u.GetPermissions() {
		if permission.Entity == entity {
			switch operation {
			case CREATE:
				return permission.Create
			case READ:
				return permission.Read
			case UPDATE:
				return permission.Update
			case DELETE:
				return permission.Delete
			}
		}
	}
	return false
}

// CheckPassword checks if the password is correct
func (u UserGeneric) CheckPassword(password string) bool {
	return u.password == password
}
