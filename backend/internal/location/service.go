package location

import (
	"context"
	"strings"

	"aegis/internal/auditlog"
	"aegis/pkg/authctx"
	"aegis/pkg/database"

	"github.com/google/uuid"
)

type Service struct {
	Repo  *Repository
	Audit auditlog.Writer
}

func validType(t database.LocationType) bool {
	switch t {
	case database.LocationTambang, database.LocationTelcoSite, database.LocationKantor,
		database.LocationGudang, database.LocationWorkshop, database.LocationLainnya:
		return true
	default:
		return false
	}
}

func (s *Service) CreateRegion(ctx context.Context, name, code string, actor authctx.Principal, ip string) (database.Region, error) {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(strings.ToUpper(code))
	if name == "" || code == "" {
		return database.Region{}, ErrValidation
	}
	row := database.Region{Name: name, Code: code}
	if err := s.Repo.CreateRegion(ctx, &row); err != nil {
		return database.Region{}, err
	}
	actx, cancel := database.AfterCommit(ctx)
	defer cancel()
	_ = s.Audit.Insert(actx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "Region", EntityID: row.ID.String(), Action: database.AuditCreated,
		After: map[string]any{"name": row.Name, "code": row.Code},
	})
	return row, nil
}

func (s *Service) ListRegions(ctx context.Context) ([]database.Region, error) {
	return s.Repo.ListRegions(ctx)
}

type LocationInput struct {
	Name         string
	Code         string
	Type         database.LocationType
	RegionID     *uuid.UUID
	SupervisorID uuid.UUID
	HSEOfficerID uuid.UUID
}

func (s *Service) CreateLocation(ctx context.Context, in LocationInput, actor authctx.Principal, ip string) (database.Location, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Code = strings.TrimSpace(strings.ToUpper(in.Code))
	if in.Name == "" || in.Code == "" || !validType(in.Type) || in.SupervisorID == uuid.Nil || in.HSEOfficerID == uuid.Nil {
		return database.Location{}, ErrValidation
	}
	ok, err := s.Repo.UserExists(ctx, in.SupervisorID)
	if err != nil || !ok {
		return database.Location{}, ErrValidation
	}
	ok, err = s.Repo.UserExists(ctx, in.HSEOfficerID)
	if err != nil || !ok {
		return database.Location{}, ErrValidation
	}
	if in.RegionID != nil {
		ok, err = s.Repo.RegionExists(ctx, *in.RegionID)
		if err != nil || !ok {
			return database.Location{}, ErrValidation
		}
	}
	row := database.Location{
		Name:         in.Name,
		Code:         in.Code,
		Type:         in.Type,
		RegionID:     in.RegionID,
		SupervisorID: in.SupervisorID,
		HSEOfficerID: in.HSEOfficerID,
		IsActive:     true,
	}
	if err := s.Repo.CreateLocation(ctx, &row); err != nil {
		return database.Location{}, err
	}
	actx, cancel := database.AfterCommit(ctx)
	defer cancel()
	_ = s.Audit.Insert(actx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "Location", EntityID: row.ID.String(), Action: database.AuditCreated,
		After: map[string]any{"name": row.Name, "code": row.Code, "type": row.Type},
	})
	return row, nil
}

func (s *Service) PatchLocation(ctx context.Context, id uuid.UUID, in LocationInput, actor authctx.Principal, ip string) (database.Location, error) {
	row, err := s.Repo.FindLocation(ctx, id)
	if err != nil {
		return database.Location{}, err
	}
	before := map[string]any{"name": row.Name, "code": row.Code, "type": row.Type, "isActive": row.IsActive}
	in.Name = strings.TrimSpace(in.Name)
	in.Code = strings.TrimSpace(strings.ToUpper(in.Code))
	if in.Name != "" {
		row.Name = in.Name
	}
	if in.Code != "" {
		row.Code = in.Code
	}
	if in.Type != "" {
		if !validType(in.Type) {
			return database.Location{}, ErrValidation
		}
		row.Type = in.Type
	}
	if in.SupervisorID != uuid.Nil {
		ok, err := s.Repo.UserExists(ctx, in.SupervisorID)
		if err != nil || !ok {
			return database.Location{}, ErrValidation
		}
		row.SupervisorID = in.SupervisorID
	}
	if in.HSEOfficerID != uuid.Nil {
		ok, err := s.Repo.UserExists(ctx, in.HSEOfficerID)
		if err != nil || !ok {
			return database.Location{}, ErrValidation
		}
		row.HSEOfficerID = in.HSEOfficerID
	}
	if in.RegionID != nil {
		ok, err := s.Repo.RegionExists(ctx, *in.RegionID)
		if err != nil || !ok {
			return database.Location{}, ErrValidation
		}
		row.RegionID = in.RegionID
	}
	if err := s.Repo.SaveLocation(ctx, &row); err != nil {
		return database.Location{}, err
	}
	actx, cancel := database.AfterCommit(ctx)
	defer cancel()
	_ = s.Audit.Insert(actx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "Location", EntityID: row.ID.String(), Action: database.AuditUpdated,
		Before: before,
		After:  map[string]any{"name": row.Name, "code": row.Code, "type": row.Type, "isActive": row.IsActive},
	})
	return row, nil
}

func (s *Service) Deactivate(ctx context.Context, id uuid.UUID, actor authctx.Principal, ip string) (database.Location, error) {
	row, err := s.Repo.FindLocation(ctx, id)
	if err != nil {
		return database.Location{}, err
	}
	before := map[string]any{"isActive": row.IsActive}
	row.IsActive = false
	if err := s.Repo.SaveLocation(ctx, &row); err != nil {
		return database.Location{}, err
	}
	actx, cancel := database.AfterCommit(ctx)
	defer cancel()
	_ = s.Audit.Insert(actx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "Location", EntityID: row.ID.String(), Action: database.AuditUpdated,
		Before: before, After: map[string]any{"isActive": false},
	})
	return row, nil
}

func (s *Service) ListLocations(ctx context.Context) ([]database.Location, error) {
	return s.Repo.ListLocations(ctx)
}

func (s *Service) CreateArea(ctx context.Context, locationID uuid.UUID, name, code string, actor authctx.Principal, ip string) (database.Area, error) {
	if _, err := s.Repo.FindLocation(ctx, locationID); err != nil {
		return database.Area{}, err
	}
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(strings.ToUpper(code))
	if name == "" || code == "" {
		return database.Area{}, ErrValidation
	}
	row := database.Area{Name: name, Code: code, LocationID: locationID}
	if err := s.Repo.CreateArea(ctx, &row); err != nil {
		return database.Area{}, err
	}
	actx, cancel := database.AfterCommit(ctx)
	defer cancel()
	_ = s.Audit.Insert(actx, auditlog.Entry{
		UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
		EntityType: "Area", EntityID: row.ID.String(), Action: database.AuditCreated,
		After: map[string]any{"name": row.Name, "code": row.Code, "locationId": locationID.String()},
	})
	return row, nil
}

func (s *Service) ListAreas(ctx context.Context, locationID uuid.UUID) ([]database.Area, error) {
	if _, err := s.Repo.FindLocation(ctx, locationID); err != nil {
		return nil, err
	}
	return s.Repo.ListAreas(ctx, locationID)
}
