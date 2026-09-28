package auditlog

import (
	"context"
	"fmt"

	"aegis/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Entry struct {
	UserID     uuid.UUID
	UserRole   string
	IPAddress  string
	EntityType string
	EntityID   string
	Action     database.AuditAction
	Before     map[string]any
	After      map[string]any
}

type Writer interface {
	Insert(ctx context.Context, e Entry) error
}

type Repository struct {
	DB *gorm.DB
}

func (r *Repository) Insert(ctx context.Context, e Entry) error {
	row := database.AuditLog{
		UserID:     e.UserID,
		UserRole:   e.UserRole,
		IPAddress:  e.IPAddress,
		EntityType: e.EntityType,
		EntityID:   e.EntityID,
		Action:     e.Action,
		Before:     e.Before,
		After:      e.After,
	}
	return r.DB.WithContext(ctx).Create(&row).Error
}

type Noop struct{}

func (Noop) Insert(context.Context, Entry) error { return nil }

func (r *Repository) Update(_ context.Context, _ uuid.UUID) error {
	return fmt.Errorf("audit log is immutable")
}

func (r *Repository) Delete(_ context.Context, _ uuid.UUID) error {
	return fmt.Errorf("audit log is immutable")
}
