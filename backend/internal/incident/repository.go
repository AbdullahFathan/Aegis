package incident

import (
	"context"
	"errors"
	"fmt"
	"time"

	"aegis/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	DB *gorm.DB
}

func (r *Repository) Create(ctx context.Context, row *database.Incident) error {
	return database.With(ctx, r.DB).Create(row).Error
}

func (r *Repository) Save(ctx context.Context, row *database.Incident) error {
	return database.With(ctx, r.DB).Save(row).Error
}

func (r *Repository) Find(ctx context.Context, id uuid.UUID) (database.Incident, error) {
	var row database.Incident
	err := database.With(ctx, r.DB).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.Incident{}, ErrNotFound
	}
	return row, err
}

func (r *Repository) FindLocation(ctx context.Context, id uuid.UUID) (database.Location, error) {
	var row database.Location
	err := database.With(ctx, r.DB).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.Location{}, ErrNotFound
	}
	return row, err
}

func (r *Repository) FindArea(ctx context.Context, id uuid.UUID) (database.Area, error) {
	var row database.Area
	err := database.With(ctx, r.DB).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.Area{}, ErrNotFound
	}
	return row, err
}

func (r *Repository) CreateWorkflowLog(ctx context.Context, row *database.IncidentWorkflowLog) error {
	return database.With(ctx, r.DB).Create(row).Error
}

func (r *Repository) ListWorkflowLogs(ctx context.Context, incidentID uuid.UUID) ([]database.IncidentWorkflowLog, error) {
	var items []database.IncidentWorkflowLog
	err := database.With(ctx, r.DB).Where("incident_id = ?", incidentID).Order("created_at ASC").Find(&items).Error
	return items, err
}

func (r *Repository) ListFiles(ctx context.Context, incidentID uuid.UUID) ([]database.IncidentFile, error) {
	var items []database.IncidentFile
	err := database.With(ctx, r.DB).Where("incident_id = ?", incidentID).Order("created_at ASC").Find(&items).Error
	return items, err
}

func (r *Repository) CreateFile(ctx context.Context, row *database.IncidentFile) error {
	return database.With(ctx, r.DB).Create(row).Error
}

func (r *Repository) FindFile(ctx context.Context, id uuid.UUID) (database.IncidentFile, error) {
	var row database.IncidentFile
	err := database.With(ctx, r.DB).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.IncidentFile{}, ErrNotFound
	}
	return row, err
}

func (r *Repository) DeleteFile(ctx context.Context, id uuid.UUID) error {
	return database.With(ctx, r.DB).Delete(&database.IncidentFile{}, "id = ?", id).Error
}

func NextIncidentNumber(tx *gorm.DB, at time.Time) (string, error) {
	at = at.UTC()
	y, m := at.Year(), int(at.Month())
	var c database.IncidentNumberCounter
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("year = ? AND month = ?", y, m).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c = database.IncidentNumberCounter{Year: y, Month: m, LastSeq: 1}
		if err := tx.Create(&c).Error; err != nil {
			err = tx.Where("year = ? AND month = ?", y, m).First(&c).Error
			if err != nil {
				return "", err
			}
			c.LastSeq++
			if err := tx.Model(&database.IncidentNumberCounter{}).
				Where("year = ? AND month = ?", y, m).
				Update("last_seq", c.LastSeq).Error; err != nil {
				return "", err
			}
		}
	} else if err != nil {
		return "", err
	} else {
		c.LastSeq++
		if err := tx.Model(&database.IncidentNumberCounter{}).
			Where("year = ? AND month = ?", y, m).
			Update("last_seq", c.LastSeq).Error; err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("INC-%04d-%02d-%04d", y, m, c.LastSeq), nil
}
