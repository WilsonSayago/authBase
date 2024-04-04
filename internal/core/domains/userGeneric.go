package domain

type EntityEnum string
type OperationEnum string

const (
	USERS EntityEnum = "USERS"
)

const (
	CREATE OperationEnum = "CREATE"
	READ   OperationEnum = "READ"
	UPDATE OperationEnum = "UPDATE"
	DELETE OperationEnum = "DELETE"
)

type UserGeneric struct {
	Id          string
	Email       string
	Password    string
	Permissions []Permission
	IsAdmin     bool
}

type Permission struct {
	Entity EntityEnum
	Create bool
	Read   bool
	Update bool
	Delete bool
}

func (u *UserGeneric) HasPermission(entity EntityEnum, operation OperationEnum) bool {
	for _, permission := range u.Permissions {
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
