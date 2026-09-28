package user

import (
	"context"
	"strings"

	"aegis/internal/auditlog"
	"aegis/internal/auth"
	"aegis/pkg/authctx"
	"aegis/pkg/database"

	"github.com/google/uuid"
)

type Service struct {
	Users *Repository
	Audit auditlog.Writer
}

type CreateInput struct {
	Email    string
	Password string
	Name     string
	Role     database.Role
	Status   database.UserStatus
}

type PatchInput struct {
	Name   *string
	Role   *database.Role
	Status *database.UserStatus
}

func validRole(r database.Role) bool {
	switch r {
	case database.RoleSuperAdmin, database.RoleAdmin, database.RoleHSEManager,
		database.RoleHSEOfficer, database.RoleSupervisor, database.RoleReporter:
		return true
	default:
		return false
	}
}

func validStatus(s database.UserStatus) bool {
	return s == database.UserStatusActive || s == database.UserStatusInactive
}

func (s *Service) Create(ctx context.Context, in CreateInput, actor authctx.Principal, ip string) (database.User, error) {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	if in.Email == "" || in.Password == "" || in.Name == "" || !validRole(in.Role) {
		return database.User{}, ErrValidation
	}
	if in.Status == "" {
		in.Status = database.UserStatusActive
	}
	if !validStatus(in.Status) {
		return database.User{}, ErrValidation
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return database.User{}, err
	}
	u := database.User{
		Email:        in.Email,
		PasswordHash: hash,
		Name:         in.Name,
		Role:         in.Role,
		Status:       in.Status,
	}
	if err := s.Users.Create(&u); err != nil {
		return database.User{}, err
	}
	_ = s.Audit.Insert(ctx, auditlog.Entry{
		UserID:     actor.ID,
		UserRole:   string(actor.Role),
		IPAddress:  ip,
		EntityType: "User",
		EntityID:   u.ID.String(),
		Action:     database.AuditCreated,
		After:      map[string]any{"email": u.Email, "role": u.Role, "status": u.Status},
	})
	return u, nil
}

func (s *Service) Patch(ctx context.Context, id uuid.UUID, in PatchInput, actor authctx.Principal, ip string) (database.User, error) {
	u, err := s.Users.FindByID(id)
	if err != nil {
		return database.User{}, err
	}
	before := map[string]any{"name": u.Name, "role": u.Role, "status": u.Status}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return database.User{}, ErrValidation
		}
		u.Name = name
	}
	if in.Role != nil {
		if !validRole(*in.Role) {
			return database.User{}, ErrValidation
		}
		u.Role = *in.Role
	}
	if in.Status != nil {
		if !validStatus(*in.Status) {
			return database.User{}, ErrValidation
		}
		u.Status = *in.Status
	}
	if err := s.Users.Update(&u); err != nil {
		return database.User{}, err
	}
	_ = s.Audit.Insert(ctx, auditlog.Entry{
		UserID:     actor.ID,
		UserRole:   string(actor.Role),
		IPAddress:  ip,
		EntityType: "User",
		EntityID:   u.ID.String(),
		Action:     database.AuditUpdated,
		Before:     before,
		After:      map[string]any{"name": u.Name, "role": u.Role, "status": u.Status},
	})
	return u, nil
}

func (s *Service) Get(id uuid.UUID) (database.User, error) {
	return s.Users.FindByID(id)
}

func (s *Service) List(page, pageSize int, role, status string) ([]database.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.Users.List(page, pageSize, role, status)
}
