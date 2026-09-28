package rbac

import "aegis/pkg/database"

type Permission string

const (
	UsersRead      Permission = "users:read"
	UsersWrite     Permission = "users:write"
	LocationsRead  Permission = "locations:read"
	LocationsWrite Permission = "locations:write"
	IncidentsClose Permission = "incidents:close"
)

var allPermissions = []Permission{
	UsersRead,
	UsersWrite,
	LocationsRead,
	LocationsWrite,
	IncidentsClose,
}

func AllPermissions() []Permission {
	out := make([]Permission, len(allPermissions))
	copy(out, allPermissions)
	return out
}

func Has(role database.Role, perm Permission) bool {
	if role == database.RoleSuperAdmin {
		for _, p := range allPermissions {
			if p == perm {
				return true
			}
		}
		return false
	}

	switch perm {
	case UsersRead, UsersWrite:
		return role == database.RoleAdmin
	case LocationsWrite:
		return role == database.RoleAdmin
	case LocationsRead:
		return true
	case IncidentsClose:
		return role == database.RoleHSEManager
	default:
		return false
	}
}
