package domain

type RoutePathEnum string

var (
	USERS RoutePathEnum
	ROLES RoutePathEnum
)

type Role struct {
	Base
	Name        string
	Routes      []RoutePathEnum
	Permissions []Permission
}
