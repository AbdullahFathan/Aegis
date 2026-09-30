package correctiveaction

import (
	"context"
	"errors"
	"strings"
	"time"

	"aegis/internal/auditlog"
	"aegis/internal/incident"
	"aegis/internal/incscope"
	"aegis/pkg/authctx"
	"aegis/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AssignNotifier interface {
	OnCAAssigned(ctx context.Context, ca database.CorrectiveAction, inc database.Incident, loc database.Location)
}

type OverdueNotifier interface {
	OnCAOverdue(ctx context.Context, ca database.CorrectiveAction, loc database.Location)
	OnCADueSoon(ctx context.Context, ca database.CorrectiveAction, loc database.Location)
}

type Service struct {
	DB     *gorm.DB
	Audit  auditlog.Writer
	Notify AssignNotifier
	Jobs   OverdueNotifier
	Once   interface {
		Claim(ctx context.Context, key string, ttl time.Duration) bool
	}
	Now func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

type CreateInput struct {
	Description string
	ActionType  database.ActionType
	Priority    database.CAPriority
	AssigneeID  uuid.UUID
	DueDate     time.Time
}

type PatchInput struct {
	Description     *string
	ActionType      *database.ActionType
	Priority        *database.CAPriority
	AssigneeID      *uuid.UUID
	DueDate         *time.Time
	Status          *database.CAStatus
	CompletionNotes *string
}

type ListFilter struct {
	Status     string
	AssigneeID *uuid.UUID
	DueFrom    *time.Time
	DueTo      *time.Time
	Priority   string
	LocationID *uuid.UUID
}

func (s *Service) ListByIncident(ctx context.Context, incidentID uuid.UUID, actor authctx.Principal) ([]database.CorrectiveAction, error) {
	inc, loc, err := s.loadIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	if !incscope.CanSee(actor, inc, loc) {
		return nil, incident.ErrNotFound
	}
	var items []database.CorrectiveAction
	err = database.With(ctx, s.DB).Where("incident_id = ?", incidentID).Order("created_at ASC").Find(&items).Error
	return items, err
}

func (s *Service) Create(ctx context.Context, incidentID uuid.UUID, in CreateInput, actor authctx.Principal, ip string) (database.CorrectiveAction, error) {
	if strings.TrimSpace(in.Description) == "" {
		return database.CorrectiveAction{}, incident.WrapValidation("description is required")
	}
	if in.AssigneeID == uuid.Nil {
		return database.CorrectiveAction{}, incident.WrapValidation("assigneeId is required")
	}
	if in.DueDate.IsZero() {
		return database.CorrectiveAction{}, incident.WrapValidation("dueDate is required")
	}
	if !validActionType(in.ActionType) || !validPriority(in.Priority) {
		return database.CorrectiveAction{}, incident.WrapValidation("invalid actionType or priority")
	}
	var assignee database.User
	if err := database.With(ctx, s.DB).First(&assignee, "id = ?", in.AssigneeID).Error; err != nil {
		return database.CorrectiveAction{}, incident.WrapValidation("assignee not found")
	}
	inc, loc, err := s.loadIncident(ctx, incidentID)
	if err != nil {
		return database.CorrectiveAction{}, err
	}
	if !incscope.CanSee(actor, inc, loc) {
		return database.CorrectiveAction{}, incident.ErrNotFound
	}
	if !s.canManage(actor, loc) {
		return database.CorrectiveAction{}, incident.ErrForbidden
	}
	if inc.Status == database.StatusClosed {
		return database.CorrectiveAction{}, incident.WrapIllegal("cannot add CA to a closed incident")
	}
	row := database.CorrectiveAction{
		IncidentID:  incidentID,
		Description: strings.TrimSpace(in.Description),
		ActionType:  in.ActionType,
		Priority:    in.Priority,
		Status:      database.CAStatusOpen,
		AssigneeID:  in.AssigneeID,
		DueDate:     in.DueDate.UTC(),
	}
	if err := database.With(ctx, s.DB).Create(&row).Error; err != nil {
		return database.CorrectiveAction{}, err
	}
	actx, cancel := database.AfterCommit(ctx)
	defer cancel()
	_ = s.Audit.Insert(actx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "CorrectiveAction", EntityID: row.ID.String(),
		Action: database.AuditCreated, After: map[string]any{"incidentId": incidentID.String(), "assigneeId": in.AssigneeID.String()},
	})
	if s.Notify != nil {
		s.Notify.OnCAAssigned(actx, row, inc, loc)
	}
	return row, nil
}

func (s *Service) Patch(ctx context.Context, id uuid.UUID, in PatchInput, actor authctx.Principal, ip string) (database.CorrectiveAction, error) {
	row, inc, loc, err := s.loadCA(ctx, id)
	if err != nil {
		return database.CorrectiveAction{}, err
	}
	if !incscope.CanSee(actor, inc, loc) {
		return database.CorrectiveAction{}, incident.ErrNotFound
	}
	if inc.Status == database.StatusClosed {
		return database.CorrectiveAction{}, incident.WrapIllegal("cannot update CA on a closed incident")
	}

	statusChange := in.Status != nil
	if statusChange && !s.canPatchStatus(actor, row) {
		return database.CorrectiveAction{}, incident.ErrForbidden
	}
	if !statusChange && !s.canManage(actor, loc) && actor.ID != row.AssigneeID {
		return database.CorrectiveAction{}, incident.ErrForbidden
	}

	before := snapshotCA(row)
	if in.Description != nil && s.canManage(actor, loc) {
		row.Description = strings.TrimSpace(*in.Description)
	}
	if in.ActionType != nil && s.canManage(actor, loc) {
		row.ActionType = *in.ActionType
	}
	if in.Priority != nil && s.canManage(actor, loc) {
		row.Priority = *in.Priority
	}
	if in.AssigneeID != nil && s.canManage(actor, loc) {
		row.AssigneeID = *in.AssigneeID
	}
	if in.DueDate != nil && s.canManage(actor, loc) {
		row.DueDate = in.DueDate.UTC()
	}
	if in.CompletionNotes != nil {
		n := strings.TrimSpace(*in.CompletionNotes)
		row.CompletionNotes = &n
	}
	if in.Status != nil {
		if err := s.applyStatus(&row, *in.Status); err != nil {
			return database.CorrectiveAction{}, err
		}
	}
	if err := database.With(ctx, s.DB).Save(&row).Error; err != nil {
		return database.CorrectiveAction{}, err
	}
	actx, cancel := database.AfterCommit(ctx)
	defer cancel()
	_ = s.Audit.Insert(actx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "CorrectiveAction", EntityID: row.ID.String(),
		Action: database.AuditUpdated, Before: before, After: snapshotCA(row),
	})
	return row, nil
}

func (s *Service) applyStatus(row *database.CorrectiveAction, to database.CAStatus) error {
	from := row.Status
	ok := false
	switch to {
	case database.CAStatusInProgress:
		ok = from == database.CAStatusOpen || from == database.CAStatusOverdue || from == database.CAStatusInProgress
	case database.CAStatusDone:
		ok = from == database.CAStatusOpen || from == database.CAStatusInProgress || from == database.CAStatusOverdue
		if ok {
			if row.CompletionNotes == nil || strings.TrimSpace(*row.CompletionNotes) == "" {
				return incident.WrapValidation("completionNotes is required when marking done")
			}
			now := s.now()
			row.CompletedAt = &now
		}
	default:
		return incident.WrapValidation("illegal CA status")
	}
	if !ok {
		return incident.WrapIllegal("illegal CA status transition")
	}
	row.Status = to
	return nil
}

func (s *Service) Verify(ctx context.Context, id uuid.UUID, actor authctx.Principal, ip string) (database.CorrectiveAction, error) {
	row, inc, loc, err := s.loadCA(ctx, id)
	if err != nil {
		return database.CorrectiveAction{}, err
	}
	if !incscope.CanSee(actor, inc, loc) {
		return database.CorrectiveAction{}, incident.ErrNotFound
	}
	if actor.Role != database.RoleHSEManager && actor.Role != database.RoleSuperAdmin {
		if actor.Role != database.RoleHSEOfficer || !incscope.IsLocationOfficer(actor, loc) {
			return database.CorrectiveAction{}, incident.ErrForbidden
		}
	}
	if row.Status != database.CAStatusDone {
		return database.CorrectiveAction{}, incident.WrapIllegal("only done CA can be verified")
	}
	now := s.now()
	row.Status = database.CAStatusVerified
	row.VerifiedByID = &actor.ID
	row.VerifiedAt = &now
	if err := database.With(ctx, s.DB).Save(&row).Error; err != nil {
		return database.CorrectiveAction{}, err
	}
	actx, cancel := database.AfterCommit(ctx)
	defer cancel()
	_ = s.Audit.Insert(actx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "CorrectiveAction", EntityID: row.ID.String(),
		Action: database.AuditUpdated, After: snapshotCA(row),
	})
	return row, nil
}

func (s *Service) Tracker(ctx context.Context, actor authctx.Principal, f ListFilter) ([]database.CorrectiveAction, error) {
	q := incscope.ApplyCAListFilter(database.With(ctx, s.DB).Session(&gorm.Session{}), actor)
	if f.Status != "" {
		q = q.Where("corrective_actions.status = ?", f.Status)
	}
	if f.AssigneeID != nil {
		q = q.Where("corrective_actions.assignee_id = ?", *f.AssigneeID)
	}
	if f.Priority != "" {
		q = q.Where("corrective_actions.priority = ?", f.Priority)
	}
	if f.LocationID != nil {
		q = q.Where("incidents.location_id = ?", *f.LocationID)
	}
	if f.DueFrom != nil {
		q = q.Where("corrective_actions.due_date >= ?", *f.DueFrom)
	}
	if f.DueTo != nil {
		q = q.Where("corrective_actions.due_date <= ?", *f.DueTo)
	}
	var items []database.CorrectiveAction
	err := q.Order("corrective_actions.due_date ASC").Find(&items).Error
	return items, err
}

func MarkOverdue(db *gorm.DB, now time.Time) ([]database.CorrectiveAction, error) {
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	var rows []database.CorrectiveAction
	err := db.Where("status IN ? AND due_date < ?", []database.CAStatus{database.CAStatusOpen, database.CAStatusInProgress}, today).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Status = database.CAStatusOverdue
		if err := db.Save(&rows[i]).Error; err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (s *Service) RunOverdueJob(ctx context.Context, now time.Time) error {
	rows, err := MarkOverdue(database.With(ctx, s.DB), now)
	if err != nil {
		return err
	}
	day := now.UTC().Format("2006-01-02")
	for _, ca := range rows {
		var inc database.Incident
		if err := database.With(ctx, s.DB).First(&inc, "id = ?", ca.IncidentID).Error; err != nil {
			continue
		}
		var loc database.Location
		if err := database.With(ctx, s.DB).First(&loc, "id = ?", inc.LocationID).Error; err != nil {
			continue
		}
		if s.Once != nil && !s.Once.Claim(ctx, "ca-overdue:"+ca.ID.String()+":"+day, 24*time.Hour) {
			continue
		}
		if s.Jobs != nil {
			s.Jobs.OnCAOverdue(ctx, ca, loc)
		}
	}
	return s.runDueSoon(ctx, now)
}

func (s *Service) runDueSoon(ctx context.Context, now time.Time) error {
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	soon := today.Add(48 * time.Hour)
	var rows []database.CorrectiveAction
	err := database.With(ctx, s.DB).Where("status IN ? AND due_date >= ? AND due_date <= ?",
		[]database.CAStatus{database.CAStatusOpen, database.CAStatusInProgress, database.CAStatusOverdue},
		today, soon).Find(&rows).Error
	if err != nil {
		return err
	}
	day := now.UTC().Format("2006-01-02")
	for _, ca := range rows {
		var inc database.Incident
		if err := database.With(ctx, s.DB).First(&inc, "id = ?", ca.IncidentID).Error; err != nil {
			continue
		}
		var loc database.Location
		if err := database.With(ctx, s.DB).First(&loc, "id = ?", inc.LocationID).Error; err != nil {
			continue
		}
		if s.Once != nil && !s.Once.Claim(ctx, "ca-due:"+ca.ID.String()+":"+day, 24*time.Hour) {
			continue
		}
		if s.Jobs != nil {
			s.Jobs.OnCADueSoon(ctx, ca, loc)
		}
	}
	return nil
}

func (s *Service) canManage(actor authctx.Principal, loc database.Location) bool {
	if actor.Role == database.RoleHSEManager || actor.Role == database.RoleSuperAdmin {
		return true
	}
	return actor.Role == database.RoleHSEOfficer && incscope.IsLocationOfficer(actor, loc)
}

func (s *Service) canPatchStatus(actor authctx.Principal, row database.CorrectiveAction) bool {
	if actor.Role == database.RoleHSEManager || actor.Role == database.RoleSuperAdmin {
		return true
	}
	return actor.ID == row.AssigneeID
}

func (s *Service) loadIncident(ctx context.Context, id uuid.UUID) (database.Incident, database.Location, error) {
	var inc database.Incident
	err := database.With(ctx, s.DB).First(&inc, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.Incident{}, database.Location{}, incident.ErrNotFound
	}
	if err != nil {
		return database.Incident{}, database.Location{}, err
	}
	var loc database.Location
	if err := database.With(ctx, s.DB).First(&loc, "id = ?", inc.LocationID).Error; err != nil {
		return database.Incident{}, database.Location{}, err
	}
	return inc, loc, nil
}

func (s *Service) loadCA(ctx context.Context, id uuid.UUID) (database.CorrectiveAction, database.Incident, database.Location, error) {
	var row database.CorrectiveAction
	err := database.With(ctx, s.DB).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.CorrectiveAction{}, database.Incident{}, database.Location{}, incident.ErrNotFound
	}
	if err != nil {
		return database.CorrectiveAction{}, database.Incident{}, database.Location{}, err
	}
	inc, loc, err := s.loadIncident(ctx, row.IncidentID)
	return row, inc, loc, err
}

func validActionType(v database.ActionType) bool {
	switch v {
	case database.ActionImmediate, database.ActionShortTerm, database.ActionLongTerm:
		return true
	}
	return false
}

func validPriority(v database.CAPriority) bool {
	switch v {
	case database.PriorityLow, database.PriorityMedium, database.PriorityHigh:
		return true
	}
	return false
}

func snapshotCA(row database.CorrectiveAction) map[string]any {
	return map[string]any{
		"status":     row.Status,
		"assigneeId": row.AssigneeID.String(),
		"dueDate":    row.DueDate,
	}
}
