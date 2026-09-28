package location

import (
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

func (r *Repository) CreateRegion(row *database.Region) error {
	err := r.DB.Create(row).Error
	if isUnique(err) {
		return ErrConflict
	}
	return err
}

func (r *Repository) ListRegions() ([]database.Region, error) {
	var items []database.Region
	err := r.DB.Order("name ASC").Find(&items).Error
	return items, err
}

func (r *Repository) CreateLocation(row *database.Location) error {
	err := r.DB.Create(row).Error
	if isUnique(err) {
		return ErrConflict
	}
	return err
}

func (r *Repository) SaveLocation(row *database.Location) error {
	err := r.DB.Save(row).Error
	if isUnique(err) {
		return ErrConflict
	}
	return err
}

func (r *Repository) FindLocation(id uuid.UUID) (database.Location, error) {
	var row database.Location
	err := r.DB.Preload("Areas").First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.Location{}, ErrNotFound
	}
	return row, err
}

func (r *Repository) ListLocations() ([]database.Location, error) {
	var items []database.Location
	err := r.DB.Preload("Areas").Order("name ASC").Find(&items).Error
	return items, err
}

func (r *Repository) UserExists(id uuid.UUID) (bool, error) {
	var n int64
	err := r.DB.Model(&database.User{}).Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

func (r *Repository) RegionExists(id uuid.UUID) (bool, error) {
	var n int64
	err := r.DB.Model(&database.Region{}).Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

func (r *Repository) CreateArea(row *database.Area) error {
	var n int64
	if err := r.DB.Model(&database.Area{}).Where("location_id = ? AND code = ?", row.LocationID, row.Code).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return ErrConflict
	}
	return r.DB.Create(row).Error
}

func (r *Repository) ListAreas(locationID uuid.UUID) ([]database.Area, error) {
	var items []database.Area
	err := r.DB.Where("location_id = ?", locationID).Order("name ASC").Find(&items).Error
	return items, err
}
