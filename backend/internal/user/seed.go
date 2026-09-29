package user

import (
	"errors"
	"strings"

	"aegis/internal/auth"
	"aegis/pkg/database"
)

// SeedSuperAdmin inserts one active SUPER_ADMIN when username and password are
// both set and that email is not already present. An existing row is left unchanged.
// created is true only when a row was inserted.
func SeedSuperAdmin(repo *Repository, username, password string) (bool, error) {
	email := strings.TrimSpace(strings.ToLower(username))
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return false, nil
	}
	if _, err := repo.FindByEmail(email); err == nil {
		return false, nil
	} else if !errors.Is(err, ErrNotFound) {
		return false, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return false, err
	}
	u := &database.User{
		Email:        email,
		PasswordHash: hash,
		Name:         "Super Admin",
		Role:         database.RoleSuperAdmin,
		Status:       database.UserStatusActive,
	}
	if err := repo.Create(u); err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
