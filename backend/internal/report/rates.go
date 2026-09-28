package report

import (
	"time"

	"aegis/internal/incident"
	"aegis/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FrequencyRate is LTIFR/TRIFR: (count × 1_000_000) / hours.
// TRIFR recordable set is MEDICAL_TREATMENT + LTI + FATALITY (not near-miss, first aid, property, environmental).
func FrequencyRate(count int64, hours float64) (float64, error) {
	if hours <= 0 {
		return 0, incident.WrapValidation("work hours must be greater than zero")
	}
	return float64(count) * 1_000_000 / hours, nil
}

func ResolveHours(db *gorm.DB, from, to time.Time, locationID *uuid.UUID) (float64, error) {
	q := db.Model(&database.WorkHours{}).
		Where("period_start <= ? AND period_end >= ?", to.UTC(), from.UTC())
	if locationID != nil {
		q = q.Where("location_id = ?", *locationID)
	} else {
		q = q.Where("location_id IS NULL")
	}
	var sum float64
	if err := q.Select("COALESCE(SUM(hours), 0)").Scan(&sum).Error; err != nil {
		return 0, err
	}
	if sum <= 0 {
		return 0, incident.WrapValidation("work hours must be greater than zero")
	}
	return sum, nil
}

func IsRecordable(c database.IncidentCategory) bool {
	switch c {
	case database.CategoryMedicalTreatment, database.CategoryLTI, database.CategoryFatality:
		return true
	default:
		return false
	}
}
