package auditlog_test

import (
	"context"
	"testing"
	"time"

	"aegis/internal/auditlog"
	"aegis/internal/auth"
	"aegis/internal/file"
	"aegis/internal/incident"
	"aegis/pkg/authctx"
	"aegis/pkg/database"
	"aegis/pkg/storage"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const longDesc = "Worker slipped on wet surface near the loading area during morning shift."

func TestSubmitStatusChangedAndFileUploaded(t *testing.T) {
	auth.SetBcryptCost(bcrypt.MinCost)
	db, err := database.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(db))

	hash, err := auth.HashPassword("password12")
	require.NoError(t, err)
	rep := database.User{Email: "rep@example.com", PasswordHash: hash, Name: "r", Role: database.RoleReporter, Status: database.UserStatusActive}
	sup := database.User{Email: "sup@example.com", PasswordHash: hash, Name: "s", Role: database.RoleSupervisor, Status: database.UserStatusActive}
	off := database.User{Email: "off@example.com", PasswordHash: hash, Name: "o", Role: database.RoleHSEOfficer, Status: database.UserStatusActive}
	require.NoError(t, db.Create(&rep).Error)
	require.NoError(t, db.Create(&sup).Error)
	require.NoError(t, db.Create(&off).Error)
	loc := database.Location{Name: "A", Code: "TMB-A", Type: database.LocationTambang, SupervisorID: sup.ID, HSEOfficerID: off.ID, IsActive: true}
	require.NoError(t, db.Create(&loc).Error)

	audit := &auditlog.Repository{DB: db}
	incSvc := &incident.Service{Repo: &incident.Repository{DB: db}, Audit: audit}
	actor := authctx.Principal{ID: rep.ID, Role: rep.Role}
	row, err := incSvc.Create(context.Background(), incident.CreateInput{
		Title: "Slip", Description: longDesc, Category: database.CategoryNearMiss,
		Severity: database.SeverityLow, IncidentDatetime: time.Now().UTC(), LocationID: loc.ID,
	}, actor, "10.0.0.1")
	require.NoError(t, err)

	fs := &file.Service{Repo: &incident.Repository{DB: db}, Store: storage.NewMemory(), Audit: audit}
	_, err = fs.Upload(context.Background(), row.ID, []file.Upload{{
		Name: "photo.jpg", Content: []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10},
	}}, actor, "10.0.0.1", nil)
	require.NoError(t, err)

	submitted, err := incSvc.Submit(context.Background(), row.ID, actor, "10.0.0.1")
	require.NoError(t, err)

	var changed database.AuditLog
	require.NoError(t, db.Where("action = ? AND entity_id = ?", database.AuditStatusChanged, submitted.ID.String()).First(&changed).Error)
	require.Equal(t, "DRAFT", changed.Before["status"])
	require.Equal(t, "PENDING_REVIEW", changed.After["status"])
	require.Equal(t, string(database.RoleReporter), changed.UserRole)
	require.Equal(t, "10.0.0.1", changed.IPAddress)

	var uploaded int64
	require.NoError(t, db.Model(&database.AuditLog{}).Where("action = ?", database.AuditFileUploaded).Count(&uploaded).Error)
	require.Equal(t, int64(1), uploaded)
}
