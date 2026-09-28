package workflow

import (
	"aegis/internal/escalation"
	"aegis/pkg/database"
)

func EscalationLevel(category database.IncidentCategory, severity database.Severity) (database.EscalationLevel, error) {
	return escalation.Level(category, severity)
}
