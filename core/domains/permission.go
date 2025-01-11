package domain

type OperationEnum string

const (
	CREATE OperationEnum = "CREATE"
	READ   OperationEnum = "READ"
	UPDATE OperationEnum = "UPDATE"
	DELETE OperationEnum = "DELETE"
)

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
