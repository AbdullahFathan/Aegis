package user_test

import (
	"testing"

	"aegis/internal/auth"
	"aegis/internal/user"
	"aegis/pkg/database"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func seedDB(t *testing.T) *gorm.DB {
	t.Helper()
	auth.SetBcryptCost(bcrypt.MinCost)
	db, err := database.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(db))
	return db
}

func TestSeedSuperAdminCreatesAndSkipsExisting(t *testing.T) {
	db := seedDB(t)
	repo := &user.Repository{DB: db}

	created, err := user.SeedSuperAdmin(repo, " SuperAdmin ", "password12")
	require.NoError(t, err)
	require.True(t, created)

	var row database.User
	require.NoError(t, db.Where("email = ?", "superadmin").First(&row).Error)
	require.Equal(t, database.RoleSuperAdmin, row.Role)
	require.Equal(t, database.UserStatusActive, row.Status)
	require.Equal(t, "Super Admin", row.Name)
	require.True(t, auth.CheckPassword(row.PasswordHash, "password12"))
	firstHash := row.PasswordHash

	created, err = user.SeedSuperAdmin(repo, "superadmin", "other-password")
	require.NoError(t, err)
	require.False(t, created)

	require.NoError(t, db.Where("email = ?", "superadmin").First(&row).Error)
	require.Equal(t, firstHash, row.PasswordHash)
	require.True(t, auth.CheckPassword(row.PasswordHash, "password12"))
}

func TestSeedSuperAdminSkipsWhenUnset(t *testing.T) {
	db := seedDB(t)
	repo := &user.Repository{DB: db}

	created, err := user.SeedSuperAdmin(repo, "", "password12")
	require.NoError(t, err)
	require.False(t, created)
	created, err = user.SeedSuperAdmin(repo, "superadmin", "  ")
	require.NoError(t, err)
	require.False(t, created)

	var n int64
	require.NoError(t, db.Model(&database.User{}).Count(&n).Error)
	require.Zero(t, n)
}
