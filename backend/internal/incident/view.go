package incident

import (
	"aegis/pkg/database"
)

func Snapshot(inc database.Incident) map[string]any {
	var num any
	if inc.IncidentNumber != nil {
		num = *inc.IncidentNumber
	}
	return map[string]any{
		"title":           inc.Title,
		"description":     inc.Description,
		"category":        inc.Category,
		"severity":        inc.Severity,
		"escalationLevel": inc.EscalationLevel,
		"status":          inc.Status,
		"incidentNumber":  num,
		"hasVictim":       inc.HasVictim,
		"pendingReviewAt": inc.PendingReviewAt,
		"closedAt":        inc.ClosedAt,
	}
}

func View(inc database.Incident) map[string]any {
	return map[string]any{
		"id":                inc.ID,
		"incidentNumber":    inc.IncidentNumber,
		"title":             inc.Title,
		"description":       inc.Description,
		"category":          inc.Category,
		"severity":          inc.Severity,
		"escalationLevel":   inc.EscalationLevel,
		"status":            inc.Status,
		"incidentDatetime":  inc.IncidentDatetime,
		"locationId":        inc.LocationID,
		"areaId":            inc.AreaID,
		"reporterId":        inc.ReporterID,
		"hasVictim":         inc.HasVictim,
		"victimName":        inc.VictimName,
		"victimPosition":    inc.VictimPosition,
		"injuryDescription": inc.InjuryDescription,
		"initialTreatment":  inc.InitialTreatment,
		"witnesses":         inc.Witnesses,
		"pendingReviewAt":   inc.PendingReviewAt,
		"closedAt":          inc.ClosedAt,
		"closedById":        inc.ClosedByID,
		"createdAt":         inc.CreatedAt,
		"updatedAt":         inc.UpdatedAt,
	}
}
