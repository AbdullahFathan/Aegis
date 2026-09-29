package report

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"time"

	"aegis/internal/incident"
	"aegis/internal/incscope"
	"aegis/internal/notification"
	"aegis/pkg/authctx"
	"aegis/pkg/database"
	aegispdf "aegis/pkg/pdf"
	"aegis/pkg/storage"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type NotifyDispatcher interface {
	Dispatch(ctx context.Context, e notification.Event)
}

type Once interface {
	Claim(ctx context.Context, key string, ttl time.Duration) bool
}

type Service struct {
	DB          *gorm.DB
	Queue       JobQueue
	Store       storage.ObjectStore
	PDF         aegispdf.Renderer
	Notify      NotifyDispatcher
	Once        Once
	Now         func() time.Time
	TZ          *time.Location
	CompanyName string
	Log         *zap.Logger
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s *Service) tz() *time.Location {
	if s.TZ != nil {
		return s.TZ
	}
	return time.UTC
}

func (s *Service) pdf() aegispdf.Renderer {
	if s.PDF != nil {
		return s.PDF
	}
	return aegispdf.Fpdf{}
}

func (s *Service) store() storage.ObjectStore {
	if s.Store != nil {
		return s.Store
	}
	return storage.NewMemory()
}

func (s *Service) queue() JobQueue {
	if s.Queue != nil {
		return s.Queue
	}
	return &MemoryQueue{}
}

func (s *Service) log() *zap.Logger {
	if s.Log != nil {
		return s.Log
	}
	return zap.NewNop()
}

type QueryFilter struct {
	From         *time.Time
	To           *time.Time
	LocationID   *uuid.UUID
	Category     string
	Confidential bool
}

func (s *Service) window(f QueryFilter) (time.Time, time.Time) {
	now := s.now()
	if f.From != nil && f.To != nil {
		return f.From.UTC(), f.To.UTC()
	}
	y, m, _ := now.UTC().Date()
	from := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	return from, now.UTC()
}

func (s *Service) incidentScope(actor authctx.Principal) *gorm.DB {
	return incscope.ApplyListFilter(s.DB, actor).Where("status <> ?", database.StatusDraft)
}

func applyIncidentFilters(q *gorm.DB, f QueryFilter, from, to time.Time) *gorm.DB {
	q = q.Where("incident_datetime >= ? AND incident_datetime <= ?", from, to)
	if f.LocationID != nil {
		q = q.Where("location_id = ?", *f.LocationID)
	}
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	return q
}

func (s *Service) UpsertWorkHours(locationID *uuid.UUID, start, end time.Time, hours float64) (database.WorkHours, error) {
	if !end.After(start) {
		return database.WorkHours{}, incident.WrapValidation("periodEnd must be after periodStart")
	}
	if hours <= 0 {
		return database.WorkHours{}, incident.WrapValidation("hours must be greater than zero")
	}
	q := s.DB.Where("period_start = ? AND period_end = ?", start.UTC(), end.UTC())
	if locationID == nil {
		q = q.Where("location_id IS NULL")
	} else {
		q = q.Where("location_id = ?", *locationID)
	}
	var row database.WorkHours
	err := q.First(&row).Error
	if err == gorm.ErrRecordNotFound {
		row = database.WorkHours{LocationID: locationID, PeriodStart: start.UTC(), PeriodEnd: end.UTC(), Hours: hours}
		return row, s.DB.Create(&row).Error
	}
	if err != nil {
		return database.WorkHours{}, err
	}
	row.Hours = hours
	return row, s.DB.Save(&row).Error
}

func (s *Service) ListWorkHours(locationID *uuid.UUID) ([]database.WorkHours, error) {
	q := s.DB.Model(&database.WorkHours{}).Order("period_start DESC")
	if locationID != nil {
		q = q.Where("location_id = ?", *locationID)
	}
	var rows []database.WorkHours
	err := q.Find(&rows).Error
	return rows, err
}

type LTIFRResult struct {
	From            time.Time `json:"from"`
	To              time.Time `json:"to"`
	LTICount        int64     `json:"ltiCount"`
	RecordableCount int64     `json:"recordableCount"`
	Hours           float64   `json:"hours"`
	LTIFR           float64   `json:"ltifr"`
	TRIFR           float64   `json:"trifr"`
}

func (s *Service) LTIFR(actor authctx.Principal, f QueryFilter) (LTIFRResult, error) {
	from, to := s.window(f)
	hours, err := ResolveHours(s.DB, from, to, f.LocationID)
	if err != nil {
		return LTIFRResult{}, err
	}
	q := applyIncidentFilters(s.incidentScope(actor), f, from, to)
	var lti, rec int64
	if err := q.Where("category = ?", database.CategoryLTI).Count(&lti).Error; err != nil {
		return LTIFRResult{}, err
	}
	q2 := applyIncidentFilters(s.incidentScope(actor), f, from, to)
	if err := q2.Where("category IN ?", []database.IncidentCategory{
		database.CategoryMedicalTreatment, database.CategoryLTI, database.CategoryFatality,
	}).Count(&rec).Error; err != nil {
		return LTIFRResult{}, err
	}
	ltifr, err := FrequencyRate(lti, hours)
	if err != nil {
		return LTIFRResult{}, err
	}
	trifr, err := FrequencyRate(rec, hours)
	if err != nil {
		return LTIFRResult{}, err
	}
	return LTIFRResult{From: from, To: to, LTICount: lti, RecordableCount: rec, Hours: hours, LTIFR: ltifr, TRIFR: trifr}, nil
}

func (s *Service) MonthlyRows(actor authctx.Principal, f QueryFilter) ([]database.Incident, error) {
	return s.listIncidents(actor, f)
}

func (s *Service) listIncidents(actor authctx.Principal, f QueryFilter) ([]database.Incident, error) {
	from, to := s.window(f)
	var items []database.Incident
	err := applyIncidentFilters(s.incidentScope(actor), f, from, to).
		Order("incident_datetime ASC").Find(&items).Error
	return items, err
}

func (s *Service) listCAs(actor authctx.Principal, f QueryFilter) ([]database.CorrectiveAction, error) {
	from, to := s.window(f)
	q := incscope.ApplyCAListFilter(s.DB, actor).
		Where("incidents.status <> ?", database.StatusDraft).
		Where("incidents.incident_datetime >= ? AND incidents.incident_datetime <= ?", from, to)
	if f.LocationID != nil {
		q = q.Where("incidents.location_id = ?", *f.LocationID)
	}
	if f.Category != "" {
		q = q.Where("incidents.category = ?", f.Category)
	}
	var items []database.CorrectiveAction
	err := q.Order("corrective_actions.due_date ASC").Find(&items).Error
	return items, err
}

func MonthlyCSV(items []database.Incident) [][]string {
	rows := [][]string{{"incidentNumber", "title", "category", "severity", "status", "locationId", "incidentDatetime"}}
	for _, it := range items {
		num := ""
		if it.IncidentNumber != nil {
			num = *it.IncidentNumber
		}
		rows = append(rows, []string{
			num, it.Title, string(it.Category), string(it.Severity), string(it.Status),
			it.LocationID.String(), it.IncidentDatetime.UTC().Format(time.RFC3339),
		})
	}
	return rows
}

func CACSV(items []database.CorrectiveAction) [][]string {
	rows := [][]string{{"id", "incidentId", "description", "status", "priority", "assigneeId", "dueDate", "completedAt"}}
	for _, it := range items {
		completed := ""
		if it.CompletedAt != nil {
			completed = it.CompletedAt.UTC().Format(time.RFC3339)
		}
		rows = append(rows, []string{
			it.ID.String(), it.IncidentID.String(), it.Description, string(it.Status), string(it.Priority),
			it.AssigneeID.String(), it.DueDate.UTC().Format("2006-01-02"), completed,
		})
	}
	return rows
}

func LTIFRCSV(r LTIFRResult) [][]string {
	return [][]string{
		{"periodFrom", "periodTo", "ltiCount", "recordableCount", "hours", "ltifr", "trifr"},
		{
			r.From.UTC().Format(time.RFC3339),
			r.To.UTC().Format(time.RFC3339),
			fmt.Sprintf("%d", r.LTICount),
			fmt.Sprintf("%d", r.RecordableCount),
			fmt.Sprintf("%g", r.Hours),
			fmt.Sprintf("%g", r.LTIFR),
			fmt.Sprintf("%g", r.TRIFR),
		},
	}
}

func EncodeCSV(rows [][]string) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.WriteAll(rows); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Service) EnqueuePDF(ctx context.Context, actor authctx.Principal, typ database.ReportType, f QueryFilter, incidentID *uuid.UUID) (database.GeneratedReport, error) {
	from, to := s.window(f)
	params := map[string]any{
		"from":         from.UTC().Format(time.RFC3339),
		"to":           to.UTC().Format(time.RFC3339),
		"confidential": f.Confidential,
	}
	if f.LocationID != nil {
		params["locationId"] = f.LocationID.String()
	}
	if f.Category != "" {
		params["category"] = f.Category
	}
	if incidentID != nil {
		params["incidentId"] = incidentID.String()
		var inc database.Incident
		var loc database.Location
		if err := s.DB.First(&inc, "id = ?", *incidentID).Error; err != nil {
			return database.GeneratedReport{}, incident.ErrNotFound
		}
		if err := s.DB.First(&loc, "id = ?", inc.LocationID).Error; err != nil {
			return database.GeneratedReport{}, incident.ErrNotFound
		}
		if !incscope.CanSee(actor, inc, loc) {
			return database.GeneratedReport{}, incident.ErrNotFound
		}
	}
	row := database.GeneratedReport{
		Type:          typ,
		Format:        database.ReportFormatPDF,
		Status:        database.ReportPending,
		Params:        params,
		RequestedByID: actor.ID,
	}
	if err := s.DB.Create(&row).Error; err != nil {
		return database.GeneratedReport{}, err
	}
	if err := s.queue().Enqueue(ctx, row.ID); err != nil {
		return database.GeneratedReport{}, err
	}
	return row, nil
}

func (s *Service) GetJob(id uuid.UUID, actor authctx.Principal) (database.GeneratedReport, string, error) {
	var row database.GeneratedReport
	if err := s.DB.First(&row, "id = ?", id).Error; err != nil {
		return database.GeneratedReport{}, "", incident.ErrNotFound
	}
	if actor.Role == database.RoleHSEOfficer && row.RequestedByID != actor.ID {
		return database.GeneratedReport{}, "", incident.ErrForbidden
	}
	url := ""
	if row.Status == database.ReportDone && row.StoredKey != nil {
		u, err := s.store().PresignGet(context.Background(), *row.StoredKey, storage.SignedURLTTL)
		if err != nil {
			return row, "", err
		}
		url = u
	}
	return row, url, nil
}

func (s *Service) ListArchive(actor authctx.Principal) ([]database.GeneratedReport, error) {
	q := s.DB.Model(&database.GeneratedReport{}).Order("created_at DESC")
	if actor.Role == database.RoleHSEOfficer {
		q = q.Where("requested_by_id = ?", actor.ID)
	}
	var rows []database.GeneratedReport
	err := q.Limit(100).Find(&rows).Error
	return rows, err
}

func (s *Service) ProcessPending(ctx context.Context, n int) error {
	ids, err := s.queue().Pop(ctx, n)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		var pending []database.GeneratedReport
		if err := s.DB.Where("status = ?", database.ReportPending).Order("created_at ASC").Limit(n).Find(&pending).Error; err != nil {
			return err
		}
		for _, p := range pending {
			ids = append(ids, p.ID)
		}
	}
	for _, id := range ids {
		if err := s.ProcessOne(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ProcessOne(ctx context.Context, id uuid.UUID) error {
	var row database.GeneratedReport
	if err := s.DB.First(&row, "id = ?", id).Error; err != nil {
		return err
	}
	if row.Status != database.ReportPending {
		return nil
	}
	body, err := s.render(ctx, row)
	if err != nil {
		msg := err.Error()
		_ = s.DB.Model(&row).Updates(map[string]any{
			"status":        database.ReportFailed,
			"error_message": msg,
			"completed_at":  s.now(),
		}).Error
		s.log().Error("report_job_failed", zap.String("jobId", id.String()), zap.String("type", string(row.Type)), zap.String("error", msg))
		return nil
	}
	sum := sha256.Sum256(append([]byte(row.ID.String()), body...))
	key := "reports/" + hex.EncodeToString(sum[:]) + ".pdf"
	if err := s.store().Put(ctx, key, bytes.NewReader(body), int64(len(body)), "application/pdf"); err != nil {
		msg := err.Error()
		dbErr := s.DB.Model(&row).Updates(map[string]any{
			"status":        database.ReportFailed,
			"error_message": msg,
			"completed_at":  s.now(),
		}).Error
		s.log().Error("report_job_failed", zap.String("jobId", id.String()), zap.String("type", string(row.Type)), zap.String("error", msg))
		return dbErr
	}
	now := s.now()
	if err := s.DB.Model(&row).Updates(map[string]any{
		"status":       database.ReportDone,
		"stored_key":   key,
		"completed_at": now,
	}).Error; err != nil {
		return err
	}
	s.log().Info("report_job_done", zap.String("jobId", id.String()), zap.String("type", string(row.Type)), zap.String("storedKey", key))
	return nil
}

func (s *Service) render(ctx context.Context, row database.GeneratedReport) ([]byte, error) {
	_ = ctx
	f := paramsToFilter(row.Params)
	actor := authctx.Principal{ID: row.RequestedByID, Role: database.RoleHSEManager}
	var u database.User
	if err := s.DB.First(&u, "id = ?", row.RequestedByID).Error; err == nil {
		actor.Role = u.Role
	}
	conf, _ := row.Params["confidential"].(bool)
	period := ""
	if f.From != nil && f.To != nil {
		period = f.From.UTC().Format("2006-01-02") + " … " + f.To.UTC().Format("2006-01-02")
	}
	doc := aegispdf.Document{
		Company:      s.CompanyName,
		Period:       period,
		Number:       row.ID.String(),
		FooterName:   "HSE Manager",
		Confidential: conf,
	}
	switch row.Type {
	case database.ReportMonthly:
		items, err := s.listIncidents(actor, f)
		if err != nil {
			return nil, err
		}
		doc.Title = "Rekap Insiden Bulanan"
		for _, it := range items {
			num := ""
			if it.IncidentNumber != nil {
				num = *it.IncidentNumber
			}
			doc.Lines = append(doc.Lines, fmt.Sprintf("%s | %s | %s | %s", num, it.Title, it.Category, it.Status))
		}
		if len(doc.Lines) == 0 {
			doc.Lines = []string{"(no incidents)"}
		}
	case database.ReportCAStatus:
		items, err := s.listCAs(actor, f)
		if err != nil {
			return nil, err
		}
		doc.Title = "Corrective Action Status"
		for _, it := range items {
			doc.Lines = append(doc.Lines, fmt.Sprintf("%s | %s | %s | due %s", it.ID, it.Status, it.Description, it.DueDate.Format("2006-01-02")))
		}
		if len(doc.Lines) == 0 {
			doc.Lines = []string{"(no corrective actions)"}
		}
	case database.ReportLTIFR:
		r, err := s.LTIFR(actor, f)
		if err != nil {
			return nil, err
		}
		doc.Title = "LTIFR & TRIFR Report"
		doc.Lines = []string{
			fmt.Sprintf("LTI count: %d", r.LTICount),
			fmt.Sprintf("Recordable count: %d", r.RecordableCount),
			fmt.Sprintf("Hours: %g", r.Hours),
			fmt.Sprintf("LTIFR: %g", r.LTIFR),
			fmt.Sprintf("TRIFR: %g", r.TRIFR),
		}
	case database.ReportInvestigation:
		raw, _ := row.Params["incidentId"].(string)
		incID, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("missing incidentId")
		}
		var inc database.Incident
		if err := s.DB.First(&inc, "id = ?", incID).Error; err != nil {
			return nil, err
		}
		doc.Title = "Incident Investigation Report"
		if inc.IncidentNumber != nil {
			doc.Number = *inc.IncidentNumber
		}
		doc.Lines = []string{
			"Title: " + inc.Title,
			"Category: " + string(inc.Category),
			"Severity: " + string(inc.Severity),
			"Description: " + inc.Description,
		}
		var rca database.RootCauseAnalysis
		if err := s.DB.Where("incident_id = ?", inc.ID).First(&rca).Error; err == nil {
			doc.Lines = append(doc.Lines, "Timeline: "+rca.Timeline, "Human factor: "+rca.HumanFactor)
		}
	default:
		return nil, fmt.Errorf("unknown report type")
	}
	return s.pdf().Render(doc)
}

func paramsToFilter(p map[string]any) QueryFilter {
	var f QueryFilter
	if p == nil {
		return f
	}
	if v, ok := p["from"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = &t
		}
	}
	if v, ok := p["to"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = &t
		}
	}
	if v, ok := p["locationId"].(string); ok {
		if id, err := uuid.Parse(v); err == nil {
			f.LocationID = &id
		}
	}
	if v, ok := p["category"].(string); ok {
		f.Category = v
	}
	if v, ok := p["confidential"].(bool); ok {
		f.Confidential = v
	}
	return f
}

func (s *Service) RunMonthlySchedule(ctx context.Context, now time.Time) error {
	loc := s.tz()
	local := now.In(loc)
	if local.Day() != 1 {
		return nil
	}
	key := fmt.Sprintf("monthly-report:%04d-%02d", local.Year(), local.Month())
	if s.Once != nil && !s.Once.Claim(ctx, key, 40*24*time.Hour) {
		return nil
	}
	prev := local.AddDate(0, -1, 0)
	from := time.Date(prev.Year(), prev.Month(), 1, 0, 0, 0, 0, loc)
	to := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc).Add(-time.Nanosecond)
	var managers []database.User
	if err := s.DB.Where("role = ? AND status = ?", database.RoleHSEManager, database.UserStatusActive).Find(&managers).Error; err != nil {
		return err
	}
	if len(managers) == 0 {
		return nil
	}
	actor := authctx.Principal{ID: managers[0].ID, Role: managers[0].Role}
	f := QueryFilter{From: &from, To: &to}
	job, err := s.EnqueuePDF(ctx, actor, database.ReportMonthly, f, nil)
	if err != nil {
		return err
	}
	if s.Notify == nil {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(managers))
	for _, m := range managers {
		ids = append(ids, m.ID)
	}
	s.Notify.Dispatch(ctx, notification.Event{
		Type:          database.NotifReportReady,
		Priority:      database.NotifInfo,
		Title:         "Rekap bulanan dijadwalkan",
		Body:          "Laporan rekap insiden bulan sebelumnya sedang digenerate.",
		ReferenceType: "generated_report",
		ReferenceID:   job.ID,
		RecipientIDs:  ids,
	})
	return nil
}
