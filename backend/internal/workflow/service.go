package workflow

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
	"aegis/pkg/notifier"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventSink interface {
	OnVerified(ctx context.Context, inc database.Incident, loc database.Location)
	OnRejected(ctx context.Context, inc database.Incident, loc database.Location)
	OnClosed(ctx context.Context, inc database.Incident, loc database.Location)
}

type Service struct {
	DB        *gorm.DB
	Audit     auditlog.Writer
	Notify    EventSink
	Emergency notifier.EmergencyNotifier
	Now       func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s *Service) Verify(ctx context.Context, id uuid.UUID, actor authctx.Principal, ip, comment string) (database.Incident, error) {
	return s.transition(ctx, id, actor, ip, comment, database.StatusPendingReview, database.StatusUnderInvestigation, database.AuditApproved, s.canVerify)
}

func (s *Service) Reject(ctx context.Context, id uuid.UUID, actor authctx.Principal, ip, comment string) (database.Incident, error) {
	if strings.TrimSpace(comment) == "" {
		return database.Incident{}, incident.WrapValidation("comment is required")
	}
	return s.transition(ctx, id, actor, ip, comment, database.StatusPendingReview, database.StatusRejected, database.AuditRejected, s.canReject)
}

func (s *Service) Close(ctx context.Context, id uuid.UUID, actor authctx.Principal, ip, comment string) (database.Incident, error) {
	inc, loc, err := s.load(ctx, id)
	if err != nil {
		return database.Incident{}, err
	}
	if !incscope.CanSee(actor, inc, loc) {
		return database.Incident{}, incident.ErrNotFound
	}
	if actor.Role != database.RoleHSEManager && actor.Role != database.RoleSuperAdmin {
		return database.Incident{}, incident.ErrForbidden
	}
	if inc.Status == database.StatusDraft {
		return database.Incident{}, incident.WrapIllegal("cannot close a draft")
	}
	if inc.Status != database.StatusUnderInvestigation && inc.Status != database.StatusCorrectiveAction {
		return database.Incident{}, incident.WrapIllegal("illegal status transition")
	}
	var pending int64
	if err := database.With(ctx, s.DB).Model(&database.CorrectiveAction{}).
		Where("incident_id = ? AND status <> ?", inc.ID, database.CAStatusVerified).
		Count(&pending).Error; err != nil {
		return database.Incident{}, err
	}
	if pending > 0 {
		return database.Incident{}, incident.WrapIllegal("all corrective actions must be verified before close")
	}
	row, err := s.apply(ctx, inc, actor, ip, comment, database.StatusClosed, database.AuditClosed, func(row *database.Incident) {
		now := s.now()
		row.ClosedAt = &now
		row.ClosedByID = &actor.ID
	})
	if err == nil && s.Notify != nil {
		actx, cancel := database.AfterCommit(ctx)
		defer cancel()
		s.Notify.OnClosed(actx, row, loc)
	}
	return row, err
}

func (s *Service) StartCorrectiveAction(ctx context.Context, id uuid.UUID, actor authctx.Principal, ip, comment string) (database.Incident, error) {
	return s.transition(ctx, id, actor, ip, comment, database.StatusUnderInvestigation, database.StatusCorrectiveAction, database.AuditStatusChanged, s.canStartCA)
}

func (s *Service) Timeline(ctx context.Context, id uuid.UUID, actor authctx.Principal) ([]database.IncidentWorkflowLog, error) {
	inc, loc, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if !incscope.CanSee(actor, inc, loc) {
		return nil, incident.ErrNotFound
	}
	var items []database.IncidentWorkflowLog
	err = database.With(ctx, s.DB).Where("incident_id = ?", id).Order("created_at ASC").Find(&items).Error
	return items, err
}

type allowFn func(actor authctx.Principal, loc database.Location) bool

func (s *Service) canVerify(actor authctx.Principal, loc database.Location) bool {
	if actor.Role == database.RoleHSEManager || actor.Role == database.RoleSuperAdmin {
		return true
	}
	return actor.Role == database.RoleSupervisor && incscope.IsLocationSupervisor(actor, loc)
}

func (s *Service) canReject(actor authctx.Principal, loc database.Location) bool {
	if actor.Role == database.RoleHSEManager || actor.Role == database.RoleSuperAdmin {
		return true
	}
	if actor.Role == database.RoleSupervisor {
		return incscope.IsLocationSupervisor(actor, loc)
	}
	if actor.Role == database.RoleHSEOfficer {
		return incscope.IsLocationOfficer(actor, loc)
	}
	return false
}

func (s *Service) canStartCA(actor authctx.Principal, loc database.Location) bool {
	if actor.Role == database.RoleHSEManager || actor.Role == database.RoleSuperAdmin {
		return true
	}
	return actor.Role == database.RoleHSEOfficer && incscope.IsLocationOfficer(actor, loc)
}

func (s *Service) transition(ctx context.Context, id uuid.UUID, actor authctx.Principal, ip, comment string, from, to database.IncidentStatus, action database.AuditAction, allow allowFn) (database.Incident, error) {
	inc, loc, err := s.load(ctx, id)
	if err != nil {
		return database.Incident{}, err
	}
	if !incscope.CanSee(actor, inc, loc) {
		return database.Incident{}, incident.ErrNotFound
	}
	if !allow(actor, loc) {
		return database.Incident{}, incident.ErrForbidden
	}
	if inc.Status != from {
		return database.Incident{}, incident.WrapIllegal("illegal status transition")
	}
	row, err := s.apply(ctx, inc, actor, ip, comment, to, action, nil)
	if err != nil {
		return database.Incident{}, err
	}
	if s.Notify != nil || (to == database.StatusUnderInvestigation && row.Category == database.CategoryFatality && s.Emergency != nil) {
		actx, cancel := database.AfterCommit(ctx)
		defer cancel()
		if s.Notify != nil {
			switch to {
			case database.StatusUnderInvestigation:
				s.Notify.OnVerified(actx, row, loc)
			case database.StatusRejected:
				s.Notify.OnRejected(actx, row, loc)
			}
		}
		if to == database.StatusUnderInvestigation && row.Category == database.CategoryFatality && s.Emergency != nil {
			_ = s.Emergency.NotifyFatality(actx, row.ID, row.Title)
		}
	}
	return row, nil
}

func (s *Service) apply(ctx context.Context, inc database.Incident, actor authctx.Principal, ip, comment string, to database.IncidentStatus, action database.AuditAction, extra func(*database.Incident)) (database.Incident, error) {
	before := incident.Snapshot(inc)
	from := inc.Status
	inc.Status = to
	if extra != nil {
		extra(&inc)
	}
	var commentPtr *string
	if strings.TrimSpace(comment) != "" {
		c := strings.TrimSpace(comment)
		commentPtr = &c
	}
	err := database.With(ctx, s.DB).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&inc).Error; err != nil {
			return err
		}
		log := database.IncidentWorkflowLog{
			IncidentID: inc.ID,
			FromStatus: from,
			ToStatus:   to,
			ActorID:    actor.ID,
			Comment:    commentPtr,
		}
		return tx.Create(&log).Error
	})
	if err != nil {
		return database.Incident{}, err
	}
	actx, cancel := database.AfterCommit(ctx)
	defer cancel()
	_ = s.Audit.Insert(actx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "Incident", EntityID: inc.ID.String(),
		Action: action, Before: before, After: incident.Snapshot(inc),
	})
	return inc, nil
}

func (s *Service) load(ctx context.Context, id uuid.UUID) (database.Incident, database.Location, error) {
	db := database.With(ctx, s.DB)
	var inc database.Incident
	err := db.First(&inc, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.Incident{}, database.Location{}, incident.ErrNotFound
	}
	if err != nil {
		return database.Incident{}, database.Location{}, err
	}
	var loc database.Location
	if err := db.First(&loc, "id = ?", inc.LocationID).Error; err != nil {
		return database.Incident{}, database.Location{}, err
	}
	return inc, loc, nil
}
