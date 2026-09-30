package location

import (
	"context"
	"errors"
	"strings"

	"aegis/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("code already exists")
	ErrValidation = errors.New("validation failed")
)

type Repository struct {
	DB *gorm.DB
}

func isUnique(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}

func (r *Repository) CreateRegion(ctx context.Context, row *database.Region) error {
	err := database.With(ctx, r.DB).Create(row).Error
	if isUnique(err) {
		return ErrConflict
	}
	return err
}

func (r *Repository) ListRegions(ctx context.Context) ([]database.Region, error) {
	var items []database.Region
	err := database.With(ctx, r.DB).Order("name ASC").Find(&items).Error
	return items, err
}

func (r *Repository) CreateLocation(ctx context.Context, row *database.Location) error {
	err := database.With(ctx, r.DB).Create(row).Error
	if isUnique(err) {
		return ErrConflict
	}
	return err
}

func (r *Repository) SaveLocation(ctx context.Context, row *database.Location) error {
	err := database.With(ctx, r.DB).Save(row).Error
	if isUnique(err) {
		return ErrConflict
	}
	return err
}

func (r *Repository) FindLocation(ctx context.Context, id uuid.UUID) (database.Location, error) {
	var row database.Location
	err := database.With(ctx, r.DB).Preload("Areas").First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.Location{}, ErrNotFound
	}
	return row, err
}

func (r *Repository) ListLocations(ctx context.Context) ([]database.Location, error) {
	var items []database.Location
	err := database.With(ctx, r.DB).Preload("Areas").Order("name ASC").Find(&items).Error
	return items, err
}

func (r *Repository) UserExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var n int64
	err := database.With(ctx, r.DB).Model(&database.User{}).Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

func (r *Repository) RegionExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var n int64
	err := database.With(ctx, r.DB).Model(&database.Region{}).Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

func (r *Repository) CreateArea(ctx context.Context, row *database.Area) error {
	var n int64
	db := database.With(ctx, r.DB)
	if err := db.Model(&database.Area{}).Where("location_id = ? AND code = ?", row.LocationID, row.Code).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return ErrConflict
	}
	return db.Create(row).Error
}

func (r *Repository) ListAreas(ctx context.Context, locationID uuid.UUID) ([]database.Area, error) {
	var items []database.Area
	err := database.With(ctx, r.DB).Where("location_id = ?", locationID).Order("name ASC").Find(&items).Error
	return items, err
}
