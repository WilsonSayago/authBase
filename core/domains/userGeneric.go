package domain

type OperationEnum string

const (
	CREATE OperationEnum = "CREATE"
	READ   OperationEnum = "READ"
	UPDATE OperationEnum = "UPDATE"
	DELETE OperationEnum = "DELETE"
)

type IUserGeneric interface {
	GetId() string
	GetEmail() string
	GetPassword() string
	GetPermissions() []Permission
	GetIsAdmin() bool
	HasPermission(entity string, operation OperationEnum) bool
}

type UserGeneric struct {
	id          string
	email       string
	password    string
	permissions []Permission
	isAdmin     bool
}

func NewUserGeneric(id string, email string, password string, permissions []Permission, isAdmin bool) UserGeneric {
	return UserGeneric{id: id, email: email, password: password, permissions: permissions, isAdmin: isAdmin}
}

func (u UserGeneric) GetId() string {
	return u.id
}

func (u UserGeneric) GetEmail() string {
	return u.email
}

func (u UserGeneric) GetPassword() string {
	return u.password
}

func (u UserGeneric) GetPermissions() []Permission {
	return u.permissions
}

func (u UserGeneric) GetIsAdmin() bool {
	return u.isAdmin
}

type Permission struct {
	Entity string
	Create bool
	Read   bool
	Update bool
	Delete bool
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
