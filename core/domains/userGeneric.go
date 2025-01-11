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
