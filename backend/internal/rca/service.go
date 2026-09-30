package rca

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

type Service struct {
	DB    *gorm.DB
	Audit auditlog.Writer
	Now   func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

type UpsertInput struct {
	Timeline          string
	HumanFactor       string
	EnvironmentFactor string
	EquipmentFactor   string
	FiveWhys          []database.FiveWhy
	Fishbone          database.Fishbone
	Completed         bool
}

func (s *Service) Get(ctx context.Context, incidentID uuid.UUID, actor authctx.Principal) (database.RootCauseAnalysis, error) {
	inc, loc, err := s.loadIncident(ctx, incidentID)
	if err != nil {
		return database.RootCauseAnalysis{}, err
	}
	if !incscope.CanSee(actor, inc, loc) {
		return database.RootCauseAnalysis{}, incident.ErrNotFound
	}
	var row database.RootCauseAnalysis
	err = database.With(ctx, s.DB).Where("incident_id = ?", incidentID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.RootCauseAnalysis{}, incident.ErrNotFound
	}
	return row, err
}

func (s *Service) Upsert(ctx context.Context, incidentID uuid.UUID, in UpsertInput, actor authctx.Principal, ip string) (database.RootCauseAnalysis, error) {
	if len(in.FiveWhys) > 5 {
		return database.RootCauseAnalysis{}, incident.WrapValidation("fiveWhys supports at most 5 entries")
	}
	inc, loc, err := s.loadIncident(ctx, incidentID)
	if err != nil {
		return database.RootCauseAnalysis{}, err
	}
	if !incscope.CanSee(actor, inc, loc) {
		return database.RootCauseAnalysis{}, incident.ErrNotFound
	}
	if !s.canWrite(actor, loc) {
		return database.RootCauseAnalysis{}, incident.ErrForbidden
	}
	if inc.Status == database.StatusClosed {
		return database.RootCauseAnalysis{}, incident.WrapIllegal("cannot update RCA after close")
	}

	var existing database.RootCauseAnalysis
	err = database.With(ctx, s.DB).Where("incident_id = ?", incidentID).First(&existing).Error
	found := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return database.RootCauseAnalysis{}, err
	}

	row := existing
	if !found {
		row = database.RootCauseAnalysis{IncidentID: incidentID}
	}
	before := snapshotRCA(row)
	row.Timeline = strings.TrimSpace(in.Timeline)
	row.HumanFactor = strings.TrimSpace(in.HumanFactor)
	row.EnvironmentFactor = strings.TrimSpace(in.EnvironmentFactor)
	row.EquipmentFactor = strings.TrimSpace(in.EquipmentFactor)
	row.FiveWhys = in.FiveWhys
	row.Fishbone = in.Fishbone
	row.InvestigatorID = actor.ID
	if in.Completed {
		now := s.now()
		row.CompletedAt = &now
	}

	if found {
		if err := database.With(ctx, s.DB).Save(&row).Error; err != nil {
			return database.RootCauseAnalysis{}, err
		}
	} else {
		if err := database.With(ctx, s.DB).Create(&row).Error; err != nil {
			return database.RootCauseAnalysis{}, err
		}
	}
	action := database.AuditUpdated
	if !found {
		action = database.AuditCreated
	}
	actx, cancel := database.AfterCommit(ctx)
	defer cancel()
	_ = s.Audit.Insert(actx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "RootCauseAnalysis", EntityID: row.ID.String(),
		Action: action, Before: before, After: snapshotRCA(row),
	})
	return row, nil
}

func (s *Service) GetTemplate(ctx context.Context, category database.IncidentCategory) (database.RCATemplate, error) {
	var row database.RCATemplate
	err := database.With(ctx, s.DB).Where("category = ?", category).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.RCATemplate{}, incident.ErrNotFound
	}
	return row, err
}

func (s *Service) PutTemplate(ctx context.Context, category database.IncidentCategory, payload database.RCATemplatePayload, actor authctx.Principal, ip string) (database.RCATemplate, error) {
	if actor.Role != database.RoleHSEOfficer && actor.Role != database.RoleHSEManager && actor.Role != database.RoleSuperAdmin {
		return database.RCATemplate{}, incident.ErrForbidden
	}
	if len(payload.FiveWhys) > 5 {
		return database.RCATemplate{}, incident.WrapValidation("fiveWhys supports at most 5 entries")
	}
	var row database.RCATemplate
	err := database.With(ctx, s.DB).Where("category = ?", category).First(&row).Error
	found := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return database.RCATemplate{}, err
	}
	if !found {
		row = database.RCATemplate{Category: category}
	}
	row.Payload = payload
	if found {
		if err := database.With(ctx, s.DB).Save(&row).Error; err != nil {
			return database.RCATemplate{}, err
		}
	} else {
		if err := database.With(ctx, s.DB).Create(&row).Error; err != nil {
			return database.RCATemplate{}, err
		}
	}
	actx, cancel := database.AfterCommit(ctx)
	defer cancel()
	_ = s.Audit.Insert(actx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "RCATemplate", EntityID: row.ID.String(),
		Action: database.AuditUpdated, After: map[string]any{"category": string(category)},
	})
	return row, nil
}

func (s *Service) canWrite(actor authctx.Principal, loc database.Location) bool {
	if actor.Role == database.RoleHSEManager || actor.Role == database.RoleSuperAdmin {
		return true
	}
	return actor.Role == database.RoleHSEOfficer && incscope.IsLocationOfficer(actor, loc)
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

func snapshotRCA(row database.RootCauseAnalysis) map[string]any {
	return map[string]any{
		"incidentId":     row.IncidentID.String(),
		"timeline":       row.Timeline,
		"investigatorId": row.InvestigatorID.String(),
		"fiveWhysCount":  len(row.FiveWhys),
		"completedAt":    row.CompletedAt,
	}
}
