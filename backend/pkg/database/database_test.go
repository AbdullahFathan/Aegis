package database_test

import (
	"context"
	"testing"
	"time"

	"aegis/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(db))
	return db
}

func TestIsTimeout(t *testing.T) {
	require.False(t, database.IsTimeout(nil))
	require.True(t, database.IsTimeout(context.DeadlineExceeded))
	require.True(t, database.IsTimeout(context.Canceled))
	require.True(t, database.IsTimeout(&pgconn.PgError{Code: "57014", Message: "canceling statement due to user request"}))
	require.False(t, database.IsTimeout(&pgconn.PgError{Code: "23505"}))
}

func TestCanceledContextDoesNotInsert(t *testing.T) {
	db := testDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := database.With(ctx, db).Create(&database.User{
		Email:        "late@example.com",
		PasswordHash: "x",
		Name:         "Late",
		Role:         database.RoleReporter,
		Status:       database.UserStatusActive,
	}).Error
	require.Error(t, err)
	require.True(t, database.IsTimeout(err))

	var n int64
	require.NoError(t, db.Model(&database.User{}).Where("email = ?", "late@example.com").Count(&n).Error)
	require.Zero(t, n)
}

func TestAutoMigrateIdempotent(t *testing.T) {
	db, err := database.OpenSQLite("file:migrate_idempotent?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(db))
	require.NoError(t, database.AutoMigrate(db))
}

func TestUniqueIncidentNumberAndLocationCode(t *testing.T) {
	db := testDB(t)

	sup := database.User{
		Email:        "sup@example.com",
		PasswordHash: "x",
		Name:         "Supervisor",
		Role:         database.RoleSupervisor,
		Status:       database.UserStatusActive,
	}
	officer := database.User{
		Email:        "hse@example.com",
		PasswordHash: "x",
		Name:         "Officer",
		Role:         database.RoleHSEOfficer,
		Status:       database.UserStatusActive,
	}
	require.NoError(t, db.Create(&sup).Error)
	require.NoError(t, db.Create(&officer).Error)

	loc := database.Location{
		Name:         "Tambang A",
		Code:         "TMB-A",
		Type:         database.LocationTambang,
		SupervisorID: sup.ID,
		HSEOfficerID: officer.ID,
		IsActive:     true,
	}
	require.NoError(t, db.Create(&loc).Error)
	err := db.Create(&database.Location{
		Name:         "Tambang A copy",
		Code:         "TMB-A",
		Type:         database.LocationTambang,
		SupervisorID: sup.ID,
		HSEOfficerID: officer.ID,
		IsActive:     true,
	}).Error
	require.Error(t, err)

	number := "INC-2026-09-0001"
	require.NoError(t, db.Create(&database.Incident{
		IncidentNumber:   &number,
		Title:            "Slip",
		Description:      "Worker slipped on wet surface near loading area during morning shift.",
		Category:         database.CategoryNearMiss,
		Severity:         database.SeverityLow,
		EscalationLevel:  database.EscalationL1,
		Status:           database.StatusDraft,
		IncidentDatetime: time.Now().UTC(),
		LocationID:       loc.ID,
		ReporterID:       sup.ID,
		HasVictim:        false,
	}).Error)

	err = db.Create(&database.Incident{
		ID:               uuid.New(),
		IncidentNumber:   &number,
		Title:            "Another",
		Description:      "Worker slipped on wet surface near loading area during morning shift.",
		Category:         database.CategoryNearMiss,
		Severity:         database.SeverityLow,
		EscalationLevel:  database.EscalationL1,
		Status:           database.StatusDraft,
		IncidentDatetime: time.Now().UTC(),
		LocationID:       loc.ID,
		ReporterID:       sup.ID,
		HasVictim:        false,
	}).Error
	require.Error(t, err)
}
