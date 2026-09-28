package incscope

import (
	"aegis/pkg/authctx"
	"aegis/pkg/database"

	"gorm.io/gorm"
)

func SeesAllDrafts(role database.Role) bool {
	return role == database.RoleAdmin || role == database.RoleSuperAdmin
}

func CanSee(actor authctx.Principal, inc database.Incident, loc database.Location) bool {
	if inc.Status == database.StatusDraft && inc.ReporterID != actor.ID && !SeesAllDrafts(actor.Role) {
		return false
	}
	switch actor.Role {
	case database.RoleSuperAdmin, database.RoleAdmin, database.RoleHSEManager:
		return true
	case database.RoleReporter:
		return inc.ReporterID == actor.ID
	case database.RoleSupervisor:
		return loc.SupervisorID == actor.ID
	case database.RoleHSEOfficer:
		return loc.HSEOfficerID == actor.ID
	default:
		return false
	}
}

func ApplyListFilter(db *gorm.DB, actor authctx.Principal) *gorm.DB {
	q := db.Model(&database.Incident{})
	switch actor.Role {
	case database.RoleReporter:
		return q.Where("reporter_id = ?", actor.ID)
	case database.RoleSupervisor:
		q = q.Where("location_id IN (?)", db.Model(&database.Location{}).Select("id").Where("supervisor_id = ?", actor.ID))
		return excludeForeignDrafts(q, actor)
	case database.RoleHSEOfficer:
		q = q.Where("location_id IN (?)", db.Model(&database.Location{}).Select("id").Where("hse_officer_id = ?", actor.ID))
		return excludeForeignDrafts(q, actor)
	case database.RoleHSEManager:
		return excludeForeignDrafts(q, actor)
	default:
		return q
	}
}

func excludeForeignDrafts(q *gorm.DB, actor authctx.Principal) *gorm.DB {
	return q.Where("status <> ? OR reporter_id = ?", database.StatusDraft, actor.ID)
}

func IsLocationOfficer(actor authctx.Principal, loc database.Location) bool {
	return loc.HSEOfficerID == actor.ID
}

func IsLocationSupervisor(actor authctx.Principal, loc database.Location) bool {
	return loc.SupervisorID == actor.ID
}

func ApplyCAListFilter(db *gorm.DB, actor authctx.Principal) *gorm.DB {
	q := db.Model(&database.CorrectiveAction{}).
		Joins("JOIN incidents ON incidents.id = corrective_actions.incident_id AND incidents.deleted_at IS NULL")
	switch actor.Role {
	case database.RoleReporter:
		return q.Where("corrective_actions.assignee_id = ?", actor.ID)
	case database.RoleSupervisor:
		return q.Where("incidents.location_id IN (?)", db.Model(&database.Location{}).Select("id").Where("supervisor_id = ?", actor.ID))
	case database.RoleHSEOfficer:
		return q.Where("incidents.location_id IN (?)", db.Model(&database.Location{}).Select("id").Where("hse_officer_id = ?", actor.ID))
	default:
		return q
	}
}
