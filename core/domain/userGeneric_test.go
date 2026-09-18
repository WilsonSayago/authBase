package domain

import (
	"testing"
)

func TestNewUserGenericAndGetters(t *testing.T) {
	t.Parallel()

	roles := []Role{{
		Base: Base{Id: "role-1", Active: true},
		Name: "editor",
	}}
	user := NewUserGeneric("user-1", "Ada", "ada@example.com", roles, true, true)

	if got := user.GetId(); got != "user-1" {
		t.Fatalf("GetId() = %q, want %q", got, "user-1")
	}
	if got := user.GetName(); got != "Ada" {
		t.Fatalf("GetName() = %q, want %q", got, "Ada")
	}
	if got := user.GetEmail(); got != "ada@example.com" {
		t.Fatalf("GetEmail() = %q, want %q", got, "ada@example.com")
	}
	if got := user.GetIsAdmin(); !got {
		t.Fatal("GetIsAdmin() = false, want true")
	}
	if !user.GetActive() {
		t.Fatal("GetActive() = false, want true")
	}
	if got := len(user.GetRole()); got != 1 {
		t.Fatalf("len(GetRole()) = %d, want 1", got)
	}
}

func TestGetPermissionsORUnionForSameEntity(t *testing.T) {
	t.Parallel()

	roles := []Role{
		{
			Base: Base{Id: "role-read", Active: true},
			Name: "reader",
			Permissions: []Permission{{
				Entity: "users",
				Read:   true,
			}},
		},
		{
			Base: Base{Id: "role-write", Active: true},
			Name: "writer",
			Permissions: []Permission{{
				Entity: "users",
				Create: true,
				Update: true,
			}},
		},
	}
	user := NewUserGeneric("user-1", "Ada", "ada@example.com", roles, false, true)

	perms := user.GetPermissions()
	if len(perms) != 1 {
		t.Fatalf("len(GetPermissions()) = %d, want 1", len(perms))
	}

	perm := perms[0]
	if perm.Entity != "users" {
		t.Fatalf("Entity = %q, want %q", perm.Entity, "users")
	}
	if !perm.Create || !perm.Read || !perm.Update || perm.Delete {
		t.Fatalf("permission flags = %+v, want create/read/update true and delete false", perm)
	}
}

func TestGetPermissionsDistinctEntities(t *testing.T) {
	t.Parallel()

	roles := []Role{{
		Base: Base{Id: "role-1", Active: true},
		Name: "multi",
		Permissions: []Permission{
			{Entity: "users", Read: true},
			{Entity: "roles", Create: true},
		},
	}}
	user := NewUserGeneric("user-1", "Ada", "ada@example.com", roles, false, true)

	perms := user.GetPermissions()
	if len(perms) != 2 {
		t.Fatalf("len(GetPermissions()) = %d, want 2", len(perms))
	}

	byEntity := make(map[string]Permission, len(perms))
	for _, perm := range perms {
		byEntity[perm.Entity] = perm
	}
	if !byEntity["users"].Read {
		t.Fatal("users.Read = false, want true")
	}
	if !byEntity["roles"].Create {
		t.Fatal("roles.Create = false, want true")
	}
}

func TestHasPermissionTable(t *testing.T) {
	t.Parallel()

	roles := []Role{{
		Base: Base{Id: "role-1", Active: true},
		Name: "limited",
		Permissions: []Permission{{
			Entity: "users",
			Read:   true,
			Update: true,
		}},
	}}
	user := NewUserGeneric("user-1", "Ada", "ada@example.com", roles, false, true)

	tests := []struct {
		name      string
		entity    string
		operation OperationEnum
		want      bool
	}{
		{name: "allowed read", entity: "users", operation: READ, want: true},
		{name: "allowed update", entity: "users", operation: UPDATE, want: true},
		{name: "denied create", entity: "users", operation: CREATE, want: false},
		{name: "missing entity", entity: "roles", operation: READ, want: false},
		{name: "unknown operation", entity: "users", operation: OperationEnum("UNKNOWN"), want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := user.HasPermission(tc.entity, tc.operation); got != tc.want {
				t.Fatalf("HasPermission(%q, %q) = %v, want %v", tc.entity, tc.operation, got, tc.want)
			}
		})
	}
}

func TestAdminAndActiveFlagsStored(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		isAdmin bool
		active  bool
	}{
		{name: "admin active", isAdmin: true, active: true},
		{name: "non-admin inactive", isAdmin: false, active: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			user := NewUserGeneric("user-1", "Ada", "ada@example.com", nil, tc.isAdmin, tc.active)
			if got := user.GetIsAdmin(); got != tc.isAdmin {
				t.Fatalf("GetIsAdmin() = %v, want %v", got, tc.isAdmin)
			}
			if got := user.GetActive(); got != tc.active {
				t.Fatalf("GetActive() = %v, want %v", got, tc.active)
			}
		})
	}
}
