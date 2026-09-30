package auth

import (
	"context"
	"errors"

	"aegis/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("user not found")

type Repository struct {
	DB *gorm.DB
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
