package incident

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"aegis/internal/auditlog"
	"aegis/internal/escalation"
	"aegis/internal/incscope"
	"aegis/pkg/authctx"
	"aegis/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	Repo  *Repository
	Audit auditlog.Writer
	Now   func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

type CreateInput struct {
	Title             string
	Description       string
	Category          database.IncidentCategory
	Severity          database.Severity
	IncidentDatetime  time.Time
	LocationID        uuid.UUID
	AreaID            *uuid.UUID
	HasVictim         bool
	VictimName        *string
	VictimPosition    *string
	InjuryDescription *string
	InitialTreatment  *string
	Witnesses         []database.Witness
}

type PatchInput struct {
	Status            *string
	Reason            *string
	Title             *string
	Description       *string
	Category          *database.IncidentCategory
	Severity          *database.Severity
	IncidentDatetime  *time.Time
	LocationID        *uuid.UUID
	AreaID            *uuid.UUID
	HasVictim         *bool
	VictimName        *string
	VictimPosition    *string
	InjuryDescription *string
	InitialTreatment  *string
	Witnesses         *[]database.Witness
}

type ListFilter struct {
	Status     string
	Category   string
	LocationID *uuid.UUID
	From       *time.Time
	To         *time.Time
	Page       int
	PageSize   int
}

func (s *Service) Create(ctx context.Context, in CreateInput, actor authctx.Principal, ip string) (database.Incident, error) {
	if err := validatePayload(in.Title, in.Description, in.Category, in.Severity, in.IncidentDatetime, in.HasVictim, in.VictimName, in.InjuryDescription); err != nil {
		return database.Incident{}, err
	}
	lvl, err := escalation.Level(in.Category, in.Severity)
	if err != nil {
		return database.Incident{}, WrapValidation(err.Error())
	}
	loc, err := s.Repo.FindLocation(in.LocationID)
	if err != nil {
		return database.Incident{}, WrapValidation("location not found")
	}
	if !loc.IsActive {
		return database.Incident{}, WrapValidation("location is inactive")
	}
	if in.AreaID != nil {
		area, err := s.Repo.FindArea(*in.AreaID)
		if err != nil || area.LocationID != in.LocationID {
			return database.Incident{}, WrapValidation("area does not belong to location")
		}
	}
	row := database.Incident{
		Title:             strings.TrimSpace(in.Title),
		Description:       strings.TrimSpace(in.Description),
		Category:          in.Category,
		Severity:          in.Severity,
		EscalationLevel:   lvl,
		Status:            database.StatusDraft,
		IncidentDatetime:  in.IncidentDatetime.UTC(),
		LocationID:        in.LocationID,
		AreaID:            in.AreaID,
		ReporterID:        actor.ID,
		HasVictim:         in.HasVictim,
		VictimName:        trimPtr(in.VictimName),
		VictimPosition:    trimPtr(in.VictimPosition),
		InjuryDescription: trimPtr(in.InjuryDescription),
		InitialTreatment:  trimPtr(in.InitialTreatment),
		Witnesses:         in.Witnesses,
	}
	if err := s.Repo.Create(&row); err != nil {
		return database.Incident{}, err
	}
	_ = s.Audit.Insert(ctx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "Incident", EntityID: row.ID.String(),
		Action: database.AuditCreated, After: Snapshot(row),
	})
	return row, nil
}

func (s *Service) Get(id uuid.UUID, actor authctx.Principal) (database.Incident, error) {
	inc, loc, err := s.load(id)
	if err != nil {
		return database.Incident{}, err
	}
	if !incscope.CanSee(actor, inc, loc) {
		return database.Incident{}, ErrNotFound
	}
	return inc, nil
}

func (s *Service) List(actor authctx.Principal, f ListFilter) ([]database.Incident, int64, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	q := incscope.ApplyListFilter(s.Repo.DB, actor)
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.LocationID != nil {
		q = q.Where("location_id = ?", *f.LocationID)
	}
	if f.From != nil {
		q = q.Where("incident_datetime >= ?", f.From.UTC())
	}
	if f.To != nil {
		q = q.Where("incident_datetime <= ?", f.To.UTC())
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []database.Incident
	err := q.Order("created_at DESC").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&items).Error
	return items, total, err
}

func (s *Service) Patch(ctx context.Context, id uuid.UUID, in PatchInput, actor authctx.Principal, ip string) (database.Incident, error) {
	if in.Status != nil {
		return database.Incident{}, WrapValidation("status cannot be set directly")
	}
	inc, loc, err := s.load(id)
	if err != nil {
		return database.Incident{}, err
	}
	if !incscope.CanSee(actor, inc, loc) {
		return database.Incident{}, ErrNotFound
	}
	if inc.Status == database.StatusClosed {
		return database.Incident{}, WrapIllegal("closed incidents are read-only")
	}

	ownerDraft := inc.ReporterID == actor.ID && (inc.Status == database.StatusDraft || inc.Status == database.StatusRejected)
	privileged := actor.Role == database.RoleHSEOfficer && incscope.IsLocationOfficer(actor, loc) ||
		actor.Role == database.RoleHSEManager || actor.Role == database.RoleSuperAdmin
	if inc.Status != database.StatusDraft && inc.Status != database.StatusRejected {
		if !privileged {
			return database.Incident{}, ErrForbidden
		}
		if in.Reason == nil || strings.TrimSpace(*in.Reason) == "" {
			return database.Incident{}, WrapValidation("reason is required to edit a submitted incident")
		}
	} else if !ownerDraft && !privileged {
		return database.Incident{}, ErrForbidden
	}

	before := Snapshot(inc)
	if in.Title != nil {
		inc.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		inc.Description = strings.TrimSpace(*in.Description)
	}
	if in.Category != nil {
		inc.Category = *in.Category
	}
	if in.Severity != nil {
		inc.Severity = *in.Severity
	}
	if in.IncidentDatetime != nil {
		inc.IncidentDatetime = in.IncidentDatetime.UTC()
	}
	if in.LocationID != nil {
		loc2, err := s.Repo.FindLocation(*in.LocationID)
		if err != nil || !loc2.IsActive {
			return database.Incident{}, WrapValidation("location not found")
		}
		inc.LocationID = *in.LocationID
		loc = loc2
	}
	if in.AreaID != nil {
		inc.AreaID = in.AreaID
	}
	if in.HasVictim != nil {
		inc.HasVictim = *in.HasVictim
	}
	if in.VictimName != nil {
		inc.VictimName = trimPtr(in.VictimName)
	}
	if in.VictimPosition != nil {
		inc.VictimPosition = trimPtr(in.VictimPosition)
	}
	if in.InjuryDescription != nil {
		inc.InjuryDescription = trimPtr(in.InjuryDescription)
	}
	if in.InitialTreatment != nil {
		inc.InitialTreatment = trimPtr(in.InitialTreatment)
	}
	if in.Witnesses != nil {
		inc.Witnesses = *in.Witnesses
	}
	if err := validatePayload(inc.Title, inc.Description, inc.Category, inc.Severity, inc.IncidentDatetime, inc.HasVictim, inc.VictimName, inc.InjuryDescription); err != nil {
		return database.Incident{}, err
	}
	if inc.AreaID != nil {
		area, err := s.Repo.FindArea(*inc.AreaID)
		if err != nil || area.LocationID != inc.LocationID {
			return database.Incident{}, WrapValidation("area does not belong to location")
		}
	}
	lvl, err := escalation.Level(inc.Category, inc.Severity)
	if err != nil {
		return database.Incident{}, WrapValidation(err.Error())
	}
	inc.EscalationLevel = lvl
	if err := s.Repo.Save(&inc); err != nil {
		return database.Incident{}, err
	}
	after := Snapshot(inc)
	if in.Reason != nil {
		after["reason"] = strings.TrimSpace(*in.Reason)
	}
	_ = s.Audit.Insert(ctx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "Incident", EntityID: inc.ID.String(),
		Action: database.AuditUpdated, Before: before, After: after,
	})
	return inc, nil
}

func (s *Service) Submit(ctx context.Context, id uuid.UUID, actor authctx.Principal, ip string) (database.Incident, error) {
	inc, _, err := s.load(id)
	if err != nil {
		return database.Incident{}, err
	}
	if inc.ReporterID != actor.ID {
		return database.Incident{}, ErrForbidden
	}
	if inc.Status != database.StatusDraft && inc.Status != database.StatusRejected {
		return database.Incident{}, ErrConflict
	}
	if err := validatePayload(inc.Title, inc.Description, inc.Category, inc.Severity, inc.IncidentDatetime, inc.HasVictim, inc.VictimName, inc.InjuryDescription); err != nil {
		return database.Incident{}, err
	}

	before := Snapshot(inc)
	now := s.now()
	from := inc.Status
	err = s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if inc.IncidentNumber == nil {
			num, err := NextIncidentNumber(tx, now)
			if err != nil {
				return err
			}
			inc.IncidentNumber = &num
		}
		inc.Status = database.StatusPendingReview
		inc.PendingReviewAt = &now
		if err := tx.Save(&inc).Error; err != nil {
			return err
		}
		log := database.IncidentWorkflowLog{
			IncidentID: inc.ID,
			FromStatus: from,
			ToStatus:   database.StatusPendingReview,
			ActorID:    actor.ID,
		}
		return tx.Create(&log).Error
	})
	if err != nil {
		return database.Incident{}, err
	}
	_ = s.Audit.Insert(ctx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "Incident", EntityID: inc.ID.String(),
		Action: database.AuditStatusChanged, Before: before, After: Snapshot(inc),
	})
	return inc, nil
}

func (s *Service) load(id uuid.UUID) (database.Incident, database.Location, error) {
	inc, err := s.Repo.Find(id)
	if err != nil {
		return database.Incident{}, database.Location{}, err
	}
	loc, err := s.Repo.FindLocation(inc.LocationID)
	if err != nil {
		return database.Incident{}, database.Location{}, err
	}
	return inc, loc, nil
}

func validatePayload(title, desc string, cat database.IncidentCategory, sev database.Severity, when time.Time, hasVictim bool, victimName, injury *string) error {
	if strings.TrimSpace(title) == "" {
		return WrapValidation("title is required")
	}
	if utf8.RuneCountInString(strings.TrimSpace(desc)) < 50 {
		return WrapValidation("description must be at least 50 characters")
	}
	if when.IsZero() {
		return WrapValidation("incidentDatetime is required")
	}
	if _, err := escalation.Level(cat, sev); err != nil {
		return WrapValidation("invalid category or severity")
	}
	if hasVictim {
		if victimName == nil || strings.TrimSpace(*victimName) == "" {
			return WrapValidation("victimName is required when hasVictim is true")
		}
		if injury == nil || strings.TrimSpace(*injury) == "" {
			return WrapValidation("injuryDescription is required when hasVictim is true")
		}
	}
	return nil
}

func trimPtr(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}
