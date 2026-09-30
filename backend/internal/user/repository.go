package user

import (
	"context"
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

func (r *Repository) Create(ctx context.Context, u *database.User) error {
	err := database.With(ctx, r.DB).Create(u).Error
	if isUnique(err) {
		return ErrEmailTaken
	}
	return err
}

func (r *Repository) Update(ctx context.Context, u *database.User) error {
	return database.With(ctx, r.DB).Save(u).Error
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (database.User, error) {
	var u database.User
	err := database.With(ctx, r.DB).Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.User{}, ErrNotFound
	}
	return u, err
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (database.User, error) {
	var u database.User
	err := database.With(ctx, r.DB).First(&u, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.User{}, ErrNotFound
	}
	return u, err
}

func (r *Repository) List(ctx context.Context, page, pageSize int, role, status string) ([]database.User, int64, error) {
	q := database.With(ctx, r.DB).Model(&database.User{})
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
