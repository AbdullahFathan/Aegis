package rbac

import "aegis/pkg/database"

type Permission string

const (
	UsersRead       Permission = "users:read"
	UsersWrite      Permission = "users:write"
	LocationsRead   Permission = "locations:read"
	LocationsWrite  Permission = "locations:write"
	IncidentsRead   Permission = "incidents:read"
	IncidentsWrite  Permission = "incidents:write"
	IncidentsVerify Permission = "incidents:verify"
	IncidentsReject Permission = "incidents:reject"
	IncidentsClose  Permission = "incidents:close"
	FilesWrite      Permission = "files:write"
)

var allPermissions = []Permission{
	UsersRead,
	UsersWrite,
	LocationsRead,
	LocationsWrite,
	IncidentsRead,
	IncidentsWrite,
	IncidentsVerify,
	IncidentsReject,
	IncidentsClose,
	FilesWrite,
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
	case LocationsRead, IncidentsRead, IncidentsWrite, FilesWrite:
		return true
	case IncidentsVerify:
		return role == database.RoleSupervisor || role == database.RoleHSEManager
	case IncidentsReject:
		return role == database.RoleSupervisor || role == database.RoleHSEOfficer || role == database.RoleHSEManager
	case IncidentsClose:
		return role == database.RoleHSEManager
	default:
		return false
	}
}
