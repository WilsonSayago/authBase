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
