package user

import (
	"errors"
	"strings"

	"aegis/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already exists")
	ErrValidation = errors.New("validation failed")
)

type Repository struct {
	DB *gorm.DB
}

func (r *Repository) Create(u *database.User) error {
	err := r.DB.Create(u).Error
	if isUnique(err) {
		return ErrEmailTaken
	}
	return err
}

func (r *Repository) Update(u *database.User) error {
	return r.DB.Save(u).Error
}

func (r *Repository) FindByID(id uuid.UUID) (database.User, error) {
	var u database.User
	err := r.DB.First(&u, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.User{}, ErrNotFound
	}
	return u, err
}

func (r *Repository) List(page, pageSize int, role, status string) ([]database.User, int64, error) {
	q := r.DB.Model(&database.User{})
	if role != "" {
		q = q.Where("role = ?", role)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []database.User
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, err
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
