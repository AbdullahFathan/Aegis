package file_test

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"aegis/internal/auditlog"
	"aegis/internal/auth"
	correctiveaction "aegis/internal/corrective_action"
	"aegis/internal/file"
	"aegis/internal/incident"
	"aegis/pkg/authctx"
	"aegis/pkg/database"
	"aegis/pkg/storage"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const longDesc = "Worker slipped on wet surface near the loading area during morning shift."

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	auth.SetBcryptCost(bcrypt.MinCost)
	db, err := database.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(db))
	return db
}

func seed(t *testing.T, db *gorm.DB) (database.Incident, authctx.Principal, *file.Service, *storage.Memory) {
	t.Helper()
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
	incSvc := &incident.Service{Repo: &incident.Repository{DB: db}, Audit: auditlog.Noop{}}
	actor := authctx.Principal{ID: rep.ID, Role: rep.Role}
	row, err := incSvc.Create(context.Background(), incident.CreateInput{
		Title: "Slip", Description: longDesc, Category: database.CategoryNearMiss,
		Severity: database.SeverityLow, IncidentDatetime: time.Now().UTC(), LocationID: loc.ID,
	}, actor, "ip")
	require.NoError(t, err)
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	mem := &storage.Memory{Now: func() time.Time { return now }}
	fs := &file.Service{Repo: &incident.Repository{DB: db}, Store: mem, Audit: &auditlog.Repository{DB: db}, Now: func() time.Time { return now }}
	return row, actor, fs, mem
}

func jpegBytes() []byte {
	return []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
}

func TestRejectFakeMIMEAndOversize(t *testing.T) {
	db := testDB(t)
	row, actor, fs, _ := seed(t, db)

	_, err := fs.Upload(context.Background(), row.ID, []file.Upload{{
		Name: "photo.jpg", Content: []byte("%PDF-1.4 fake"),
	}}, actor, "ip", nil)
	require.ErrorIs(t, err, incident.ErrValidation)

	big := bytes.Repeat([]byte{0xFF, 0xD8, 0xFF}, int(storage.MaxBytes/3)+10)
	_, err = fs.Upload(context.Background(), row.ID, []file.Upload{{
		Name: "photo.jpg", Content: big,
	}}, actor, "ip", nil)
	require.ErrorIs(t, err, incident.ErrValidation)
}

func TestDeleteAfterSubmitRejectedAndPresignExpiry(t *testing.T) {
	db := testDB(t)
	row, actor, fs, mem := seed(t, db)
	created, err := fs.Upload(context.Background(), row.ID, []file.Upload{{
		Name: "photo.jpg", Content: jpegBytes(),
	}}, actor, "ip", nil)
	require.NoError(t, err)
	require.Len(t, created, 1)
	require.True(t, strings.HasPrefix(created[0].StoredKey, "incidents/"))
	require.NotContains(t, created[0].StoredKey, "photo.jpg")

	listed, err := fs.List(context.Background(), row.ID, actor)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	url, _ := listed[0]["url"].(string)
	require.Contains(t, url, "exp=")
	exp := time.Date(2026, 9, 28, 13, 0, 0, 0, time.UTC).Unix()
	require.Contains(t, url, "exp="+strconv.FormatInt(exp, 10))
	_, ok := mem.Get(created[0].StoredKey)
	require.True(t, ok)

	incSvc := &incident.Service{Repo: &incident.Repository{DB: db}, Audit: auditlog.Noop{}}
	_, err = incSvc.Submit(context.Background(), row.ID, actor, "ip")
	require.NoError(t, err)

	err = fs.Delete(row.ID, created[0].ID, actor)
	require.ErrorIs(t, err, incident.ErrIllegal)
}

func TestUploadCACompletion(t *testing.T) {
	db := testDB(t)
	row, actor, fs, _ := seed(t, db)
	var off database.User
	require.NoError(t, db.Where("email = ?", "off@example.com").First(&off).Error)
	incSvc := &incident.Service{Repo: &incident.Repository{DB: db}, Audit: auditlog.Noop{}}
	submitted, err := incSvc.Submit(context.Background(), row.ID, actor, "ip")
	require.NoError(t, err)
	caSvc := &correctiveaction.Service{DB: db, Audit: auditlog.Noop{}}
	ca, err := caSvc.Create(context.Background(), submitted.ID, correctiveaction.CreateInput{
		Description: "photo evidence", ActionType: database.ActionImmediate, Priority: database.PriorityLow,
		AssigneeID: actor.ID, DueDate: time.Now().UTC().Add(24 * time.Hour),
	}, authctx.Principal{ID: off.ID, Role: off.Role}, "ip")
	require.NoError(t, err)
	id := ca.ID
	created, err := fs.Upload(context.Background(), submitted.ID, []file.Upload{{
		Name: "photo.jpg", Content: jpegBytes(),
	}}, actor, "ip", &id)
	require.NoError(t, err)
	require.Equal(t, database.FileCACompletion, created[0].Context)
	require.NotNil(t, created[0].CorrectiveActionID)
	require.Equal(t, ca.ID, *created[0].CorrectiveActionID)
}
