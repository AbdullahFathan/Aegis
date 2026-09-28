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
	RCAWrite        Permission = "rca:write"
	CAWrite         Permission = "corrective_actions:write"
	CAVerify        Permission = "corrective_actions:verify"
	DashboardRead   Permission = "dashboard:read"
	ReportsExport   Permission = "reports:export"
	AuditLogsRead   Permission = "audit_logs:read"
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
	RCAWrite,
	CAWrite,
	CAVerify,
	DashboardRead,
	ReportsExport,
	AuditLogsRead,
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
	case RCAWrite, CAWrite, CAVerify:
		return role == database.RoleHSEOfficer || role == database.RoleHSEManager
	case DashboardRead:
		return role == database.RoleAdmin || role == database.RoleHSEManager
	case ReportsExport:
		return role == database.RoleAdmin || role == database.RoleHSEManager || role == database.RoleHSEOfficer
	case AuditLogsRead:
		return role == database.RoleAdmin
	default:
		return false
	}
}
