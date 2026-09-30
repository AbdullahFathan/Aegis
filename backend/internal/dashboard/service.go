package dashboard

import (
	"context"
	"time"

	"aegis/internal/report"
	"aegis/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	DB  *gorm.DB
	Now func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

type Filter struct {
	From       *time.Time
	To         *time.Time
	LocationID *uuid.UUID
	Category   string
}

type Summary struct {
	ThisMonthCount int64    `json:"thisMonthCount"`
	LastMonthCount int64    `json:"lastMonthCount"`
	Delta          int64    `json:"delta"`
	Trend          string   `json:"trend"`
	CAOverdue      int64    `json:"caOverdueCount"`
	Pipeline       Pipeline `json:"pipeline"`
	LTIFR          *float64 `json:"ltifr"`
	TRIFR          *float64 `json:"trifr"`
	WorkHours      *float64 `json:"workHours"`
}

type Pipeline struct {
	PendingReview      int64 `json:"PENDING_REVIEW"`
	UnderInvestigation int64 `json:"UNDER_INVESTIGATION"`
	CorrectiveAction   int64 `json:"CORRECTIVE_ACTION"`
	Closed             int64 `json:"CLOSED"`
}

type MonthBucket struct {
	Year  int   `json:"year"`
	Month int   `json:"month"`
	Count int64 `json:"count"`
}

type HeatCell struct {
	LocationID   string `json:"locationId"`
	LocationCode string `json:"locationCode"`
	Category     string `json:"category"`
	Count        int64  `json:"count"`
}

func (s *Service) window(f Filter) (from, to time.Time) {
	now := s.now()
	if f.From != nil && f.To != nil {
		return f.From.UTC(), f.To.UTC()
	}
	y, m, _ := now.UTC().Date()
	from = time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	to = now.UTC()
	return from, to
}

func (s *Service) incidentQ(ctx context.Context, f Filter, from, to time.Time) *gorm.DB {
	q := database.With(ctx, s.DB).Model(&database.Incident{}).Where("status <> ?", database.StatusDraft)
	if f.LocationID != nil {
		q = q.Where("location_id = ?", *f.LocationID)
	}
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	q = q.Where("incident_datetime >= ? AND incident_datetime <= ?", from, to)
	return q
}

func (s *Service) Summary(ctx context.Context, f Filter) (Summary, error) {
	from, to := s.window(f)
	monthStart := time.Date(from.UTC().Year(), from.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	prevFrom := monthStart.AddDate(0, -1, 0)
	prevTo := monthStart.Add(-time.Nanosecond)

	thisN, err := s.countIncidents(ctx, f, from, to)
	if err != nil {
		return Summary{}, err
	}
	lastN, err := s.countIncidents(ctx, f, prevFrom, prevTo)
	if err != nil {
		return Summary{}, err
	}
	trend := "flat"
	if thisN > lastN {
		trend = "up"
	} else if thisN < lastN {
		trend = "down"
	}

	pipe, err := s.pipeline(ctx, f, from, to)
	if err != nil {
		return Summary{}, err
	}
	overdue, err := s.caOverdue(ctx, f, from, to)
	if err != nil {
		return Summary{}, err
	}

	out := Summary{
		ThisMonthCount: thisN,
		LastMonthCount: lastN,
		Delta:          thisN - lastN,
		Trend:          trend,
		CAOverdue:      overdue,
		Pipeline:       pipe,
	}

	hours, herr := report.ResolveHours(database.With(ctx, s.DB), from, to, f.LocationID)
	if herr == nil {
		lti, err := s.countCategory(ctx, f, from, to, database.CategoryLTI)
		if err != nil {
			return Summary{}, err
		}
		rec, err := s.countRecordable(ctx, f, from, to)
		if err != nil {
			return Summary{}, err
		}
		ltifr, _ := report.FrequencyRate(lti, hours)
		trifr, _ := report.FrequencyRate(rec, hours)
		out.LTIFR = &ltifr
		out.TRIFR = &trifr
		out.WorkHours = &hours
	}
	return out, nil
}

func (s *Service) countIncidents(ctx context.Context, f Filter, from, to time.Time) (int64, error) {
	var n int64
	err := s.incidentQ(ctx, f, from, to).Count(&n).Error
	return n, err
}

func (s *Service) countCategory(ctx context.Context, f Filter, from, to time.Time, cat database.IncidentCategory) (int64, error) {
	var n int64
	err := s.incidentQ(ctx, f, from, to).Where("category = ?", cat).Count(&n).Error
	return n, err
}

func (s *Service) countRecordable(ctx context.Context, f Filter, from, to time.Time) (int64, error) {
	var n int64
	err := s.incidentQ(ctx, f, from, to).Where("category IN ?", []database.IncidentCategory{
		database.CategoryMedicalTreatment, database.CategoryLTI, database.CategoryFatality,
	}).Count(&n).Error
	return n, err
}

func (s *Service) pipeline(ctx context.Context, f Filter, from, to time.Time) (Pipeline, error) {
	type row struct {
		Status database.IncidentStatus
		N      int64
	}
	var rows []row
	err := s.incidentQ(ctx, f, from, to).Select("status, count(*) as n").Group("status").Scan(&rows).Error
	if err != nil {
		return Pipeline{}, err
	}
	var p Pipeline
	for _, r := range rows {
		switch r.Status {
		case database.StatusPendingReview:
			p.PendingReview = r.N
		case database.StatusUnderInvestigation:
			p.UnderInvestigation = r.N
		case database.StatusCorrectiveAction:
			p.CorrectiveAction = r.N
		case database.StatusClosed:
			p.Closed = r.N
		}
	}
	return p, nil
}

func (s *Service) caOverdue(ctx context.Context, f Filter, from, to time.Time) (int64, error) {
	q := database.With(ctx, s.DB).Model(&database.CorrectiveAction{}).
		Joins("JOIN incidents ON incidents.id = corrective_actions.incident_id AND incidents.deleted_at IS NULL").
		Where("corrective_actions.status = ?", database.CAStatusOverdue).
		Where("incidents.status <> ?", database.StatusDraft).
		Where("incidents.incident_datetime >= ? AND incidents.incident_datetime <= ?", from, to)
	if f.LocationID != nil {
		q = q.Where("incidents.location_id = ?", *f.LocationID)
	}
	if f.Category != "" {
		q = q.Where("incidents.category = ?", f.Category)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (s *Service) Trends(ctx context.Context, f Filter) ([]MonthBucket, error) {
	_, end := s.window(f)
	endMonth := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	out := make([]MonthBucket, 0, 12)
	for i := 11; i >= 0; i-- {
		start := endMonth.AddDate(0, -i, 0)
		next := start.AddDate(0, 1, 0)
		to := next.Add(-time.Nanosecond)
		n, err := s.countIncidents(ctx, f, start, to)
		if err != nil {
			return nil, err
		}
		out = append(out, MonthBucket{Year: start.Year(), Month: int(start.Month()), Count: n})
	}
	return out, nil
}

func (s *Service) Heatmap(ctx context.Context, f Filter) ([]HeatCell, error) {
	from, to := s.window(f)
	type row struct {
		LocationID uuid.UUID
		Code       string
		Category   database.IncidentCategory
		N          int64
	}
	q := database.With(ctx, s.DB).Model(&database.Incident{}).
		Select("incidents.location_id as location_id, locations.code as code, incidents.category as category, count(*) as n").
		Joins("JOIN locations ON locations.id = incidents.location_id").
		Where("incidents.status <> ?", database.StatusDraft).
		Where("incidents.incident_datetime >= ? AND incidents.incident_datetime <= ?", from, to)
	if f.LocationID != nil {
		q = q.Where("incidents.location_id = ?", *f.LocationID)
	}
	if f.Category != "" {
		q = q.Where("incidents.category = ?", f.Category)
	}
	var rows []row
	if err := q.Group("incidents.location_id, locations.code, incidents.category").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]HeatCell, 0, len(rows))
	for _, r := range rows {
		out = append(out, HeatCell{
			LocationID:   r.LocationID.String(),
			LocationCode: r.Code,
			Category:     string(r.Category),
			Count:        r.N,
		})
	}
	return out, nil
}
