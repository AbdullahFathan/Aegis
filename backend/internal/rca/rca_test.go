package rca_test

import (
	"context"
	"testing"
	"time"

	"aegis/internal/auditlog"
	"aegis/internal/auth"
	"aegis/internal/incident"
	"aegis/internal/rca"
	"aegis/pkg/authctx"
	"aegis/pkg/database"

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

func seed(t *testing.T, db *gorm.DB) (database.Incident, database.User, database.User, database.User) {
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
	row, err := incSvc.Create(context.Background(), incident.CreateInput{
		Title: "Slip", Description: longDesc, Category: database.CategoryNearMiss,
		Severity: database.SeverityLow, IncidentDatetime: time.Now().UTC(), LocationID: loc.ID,
	}, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.NoError(t, err)
	row, err = incSvc.Submit(context.Background(), row.ID, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.NoError(t, err)
	return row, rep, sup, off
}

func TestMaxFiveWhysAndUniquePerIncident(t *testing.T) {
	db := testDB(t)
	row, _, _, off := seed(t, db)
	svc := &rca.Service{DB: db, Audit: auditlog.Noop{}}
	actor := authctx.Principal{ID: off.ID, Role: off.Role}

	_, err := svc.Upsert(context.Background(), row.ID, rca.UpsertInput{
		Timeline: "t",
		FiveWhys: []database.FiveWhy{{}, {}, {}, {}, {}, {}},
	}, actor, "ip")
	require.ErrorIs(t, err, incident.ErrValidation)

	first, err := svc.Upsert(context.Background(), row.ID, rca.UpsertInput{Timeline: "one"}, actor, "ip")
	require.NoError(t, err)
	second, err := svc.Upsert(context.Background(), row.ID, rca.UpsertInput{Timeline: "two"}, actor, "ip")
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, "two", second.Timeline)

	var count int64
	require.NoError(t, db.Model(&database.RootCauseAnalysis{}).Where("incident_id = ?", row.ID).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestSupervisorForbidden(t *testing.T) {
	db := testDB(t)
	row, _, sup, _ := seed(t, db)
	svc := &rca.Service{DB: db, Audit: auditlog.Noop{}}
	_, err := svc.Upsert(context.Background(), row.ID, rca.UpsertInput{Timeline: "t"}, authctx.Principal{ID: sup.ID, Role: sup.Role}, "ip")
	require.ErrorIs(t, err, incident.ErrForbidden)
}

func TestTemplatePerCategory(t *testing.T) {
	db := testDB(t)
	_, _, _, off := seed(t, db)
	svc := &rca.Service{DB: db, Audit: auditlog.Noop{}}
	actor := authctx.Principal{ID: off.ID, Role: off.Role}

	_, err := svc.PutTemplate(context.Background(), database.CategoryNearMiss, database.RCATemplatePayload{Timeline: "nm"}, actor, "ip")
	require.NoError(t, err)
	_, err = svc.PutTemplate(context.Background(), database.CategoryLTI, database.RCATemplatePayload{Timeline: "lti"}, actor, "ip")
	require.NoError(t, err)

	nm, err := svc.GetTemplate(database.CategoryNearMiss)
	require.NoError(t, err)
	lti, err := svc.GetTemplate(database.CategoryLTI)
	require.NoError(t, err)
	require.Equal(t, "nm", nm.Payload.Timeline)
	require.Equal(t, "lti", lti.Payload.Timeline)
}

func TestClosedIncidentRejectsRCA(t *testing.T) {
	db := testDB(t)
	row, _, _, off := seed(t, db)
	require.NoError(t, db.Model(&row).Update("status", database.StatusClosed).Error)
	svc := &rca.Service{DB: db, Audit: auditlog.Noop{}}
	_, err := svc.Upsert(context.Background(), row.ID, rca.UpsertInput{Timeline: "t"}, authctx.Principal{ID: off.ID, Role: off.Role}, "ip")
	require.ErrorIs(t, err, incident.ErrIllegal)
}
