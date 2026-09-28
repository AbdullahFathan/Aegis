package escalation

import (
	"fmt"

	"aegis/pkg/database"
)

// Level implements PRD FR-IR-07.
// FATALITY is always L3 (including severity cells marked "—" in the table).
// PROPERTY_DAMAGE and ENVIRONMENTAL use the Near-Miss column.
func Level(category database.IncidentCategory, severity database.Severity) (database.EscalationLevel, error) {
	if category == database.CategoryFatality {
		if !validSeverity(severity) {
			return "", fmt.Errorf("invalid severity")
		}
		return database.EscalationL3, nil
	}

	col := category
	switch category {
	case database.CategoryPropertyDamage, database.CategoryEnvironmental:
		col = database.CategoryNearMiss
	}

	matrix := map[database.IncidentCategory]map[database.Severity]database.EscalationLevel{
		database.CategoryNearMiss: {
			database.SeverityLow:      database.EscalationL1,
			database.SeverityMedium:   database.EscalationL1,
			database.SeverityHigh:     database.EscalationL2,
			database.SeverityCritical: database.EscalationL2,
		},
		database.CategoryFirstAid: {
			database.SeverityLow:      database.EscalationL1,
			database.SeverityMedium:   database.EscalationL2,
			database.SeverityHigh:     database.EscalationL2,
			database.SeverityCritical: database.EscalationL3,
		},
		database.CategoryMedicalTreatment: {
			database.SeverityLow:      database.EscalationL2,
			database.SeverityMedium:   database.EscalationL2,
			database.SeverityHigh:     database.EscalationL3,
			database.SeverityCritical: database.EscalationL3,
		},
		database.CategoryLTI: {
			database.SeverityLow:      database.EscalationL2,
			database.SeverityMedium:   database.EscalationL3,
			database.SeverityHigh:     database.EscalationL3,
			database.SeverityCritical: database.EscalationL3,
		},
	}

	bySev, ok := matrix[col]
	if !ok {
		return "", fmt.Errorf("invalid category")
	}
	lvl, ok := bySev[severity]
	if !ok {
		return "", fmt.Errorf("invalid severity")
	}
	return lvl, nil
}

func validSeverity(s database.Severity) bool {
	switch s {
	case database.SeverityLow, database.SeverityMedium, database.SeverityHigh, database.SeverityCritical:
		return true
	default:
		return false
	}
}
