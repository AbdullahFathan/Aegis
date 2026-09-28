package correctiveaction_test

import (
	"context"
	"testing"
	"time"

	"aegis/internal/auditlog"
	"aegis/internal/auth"
	correctiveaction "aegis/internal/corrective_action"
	"aegis/internal/incident"
	"aegis/internal/notification"
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

func seed(t *testing.T, db *gorm.DB) (database.Incident, database.User, database.User, database.User, database.Location) {
	t.Helper()
	hash, err := auth.HashPassword("password12")
	require.NoError(t, err)
	rep := database.User{Email: "rep@example.com", PasswordHash: hash, Name: "r", Role: database.RoleReporter, Status: database.UserStatusActive}
	sup := database.User{Email: "sup@example.com", PasswordHash: hash, Name: "s", Role: database.RoleSupervisor, Status: database.UserStatusActive}
	off := database.User{Email: "off@example.com", PasswordHash: hash, Name: "o", Role: database.RoleHSEOfficer, Status: database.UserStatusActive}
	off2 := database.User{Email: "off2@example.com", PasswordHash: hash, Name: "o2", Role: database.RoleHSEOfficer, Status: database.UserStatusActive}
	require.NoError(t, db.Create(&rep).Error)
	require.NoError(t, db.Create(&sup).Error)
	require.NoError(t, db.Create(&off).Error)
	require.NoError(t, db.Create(&off2).Error)
	loc := database.Location{Name: "A", Code: "TMB-A", Type: database.LocationTambang, SupervisorID: sup.ID, HSEOfficerID: off.ID, IsActive: true}
	require.NoError(t, db.Create(&loc).Error)
	locB := database.Location{Name: "B", Code: "TMB-B", Type: database.LocationTambang, SupervisorID: sup.ID, HSEOfficerID: off2.ID, IsActive: true}
	require.NoError(t, db.Create(&locB).Error)
	incSvc := &incident.Service{Repo: &incident.Repository{DB: db}, Audit: auditlog.Noop{}}
	row, err := incSvc.Create(context.Background(), incident.CreateInput{
		Title: "Slip", Description: longDesc, Category: database.CategoryNearMiss,
		Severity: database.SeverityLow, IncidentDatetime: time.Now().UTC(), LocationID: loc.ID,
	}, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.NoError(t, err)
	row, err = incSvc.Submit(context.Background(), row.ID, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.NoError(t, err)
	_ = locB
	return row, rep, off, off2, loc
}

func TestAssigneePatchAndDoneNotes(t *testing.T) {
	db := testDB(t)
	inc, rep, off, _, _ := seed(t, db)
	svc := &correctiveaction.Service{DB: db, Audit: auditlog.Noop{}}
	offP := authctx.Principal{ID: off.ID, Role: off.Role}
	ca, err := svc.Create(context.Background(), inc.ID, correctiveaction.CreateInput{
		Description: "Guard rail", ActionType: database.ActionImmediate, Priority: database.PriorityHigh,
		AssigneeID: rep.ID, DueDate: time.Now().UTC().Add(48 * time.Hour),
	}, offP, "ip")
	require.NoError(t, err)

	status := database.CAStatusDone
	_, err = svc.Patch(context.Background(), ca.ID, correctiveaction.PatchInput{Status: &status}, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.ErrorIs(t, err, incident.ErrValidation)

	notes := "installed rail"
	inProgress := database.CAStatusInProgress
	_, err = svc.Patch(context.Background(), ca.ID, correctiveaction.PatchInput{Status: &inProgress}, authctx.Principal{ID: off.ID, Role: off.Role}, "ip")
	require.ErrorIs(t, err, incident.ErrForbidden)

	_, err = svc.Patch(context.Background(), ca.ID, correctiveaction.PatchInput{Status: &inProgress}, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.NoError(t, err)

	done, err := svc.Patch(context.Background(), ca.ID, correctiveaction.PatchInput{Status: &status, CompletionNotes: &notes}, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.NoError(t, err)
	require.Equal(t, database.CAStatusDone, done.Status)
	require.NotNil(t, done.CompletedAt)
}

func TestVerifyReporterForbidden(t *testing.T) {
	db := testDB(t)
	inc, rep, off, _, _ := seed(t, db)
	svc := &correctiveaction.Service{DB: db, Audit: auditlog.Noop{}}
	ca, err := svc.Create(context.Background(), inc.ID, correctiveaction.CreateInput{
		Description: "Guard rail", ActionType: database.ActionImmediate, Priority: database.PriorityHigh,
		AssigneeID: rep.ID, DueDate: time.Now().UTC().Add(48 * time.Hour),
	}, authctx.Principal{ID: off.ID, Role: off.Role}, "ip")
	require.NoError(t, err)
	notes := "done"
	st := database.CAStatusDone
	_, err = svc.Patch(context.Background(), ca.ID, correctiveaction.PatchInput{Status: &st, CompletionNotes: &notes}, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.NoError(t, err)

	_, err = svc.Verify(context.Background(), ca.ID, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.ErrorIs(t, err, incident.ErrForbidden)

	verified, err := svc.Verify(context.Background(), ca.ID, authctx.Principal{ID: off.ID, Role: off.Role}, "ip")
	require.NoError(t, err)
	require.Equal(t, database.CAStatusVerified, verified.Status)
	require.NotNil(t, verified.VerifiedByID)
	require.Equal(t, off.ID, *verified.VerifiedByID)
}

func TestOverdueClock(t *testing.T) {
	db := testDB(t)
	inc, rep, off, _, _ := seed(t, db)
	svc := &correctiveaction.Service{DB: db, Audit: auditlog.Noop{}}
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	past, err := svc.Create(context.Background(), inc.ID, correctiveaction.CreateInput{
		Description: "past", ActionType: database.ActionImmediate, Priority: database.PriorityLow,
		AssigneeID: rep.ID, DueDate: now.Add(-24 * time.Hour),
	}, authctx.Principal{ID: off.ID, Role: off.Role}, "ip")
	require.NoError(t, err)
	future, err := svc.Create(context.Background(), inc.ID, correctiveaction.CreateInput{
		Description: "future", ActionType: database.ActionImmediate, Priority: database.PriorityLow,
		AssigneeID: rep.ID, DueDate: now.Add(24 * time.Hour),
	}, authctx.Principal{ID: off.ID, Role: off.Role}, "ip")
	require.NoError(t, err)

	updated, err := correctiveaction.MarkOverdue(db, now)
	require.NoError(t, err)
	require.Len(t, updated, 1)
	require.Equal(t, past.ID, updated[0].ID)

	var fresh database.CorrectiveAction
	require.NoError(t, db.First(&fresh, "id = ?", future.ID).Error)
	require.Equal(t, database.CAStatusOpen, fresh.Status)
}

func TestRunOverdueJobNotifies(t *testing.T) {
	db := testDB(t)
	inc, rep, off, _, _ := seed(t, db)
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	svc := &correctiveaction.Service{
		DB: db, Audit: auditlog.Noop{},
		Jobs: &notification.Service{DB: db, Once: &notification.MemoryOnce{}},
		Once: &notification.MemoryOnce{},
	}
	actor := authctx.Principal{ID: off.ID, Role: off.Role}
	_, err := svc.Create(context.Background(), inc.ID, correctiveaction.CreateInput{
		Description: "past", ActionType: database.ActionImmediate, Priority: database.PriorityLow,
		AssigneeID: rep.ID, DueDate: now.Add(-24 * time.Hour),
	}, actor, "ip")
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), inc.ID, correctiveaction.CreateInput{
		Description: "soon", ActionType: database.ActionImmediate, Priority: database.PriorityLow,
		AssigneeID: rep.ID, DueDate: now.Add(24 * time.Hour),
	}, actor, "ip")
	require.NoError(t, err)
	require.NoError(t, svc.RunOverdueJob(context.Background(), now))
	require.NoError(t, svc.RunOverdueJob(context.Background(), now))
}

func TestTrackerHidesOtherSite(t *testing.T) {
	db := testDB(t)
	incA, rep, off, off2, loc := seed(t, db)
	_ = loc
	var locB database.Location
	require.NoError(t, db.Where("code = ?", "TMB-B").First(&locB).Error)
	incB := database.Incident{
		Title: "Other", Description: longDesc, Category: database.CategoryNearMiss, Severity: database.SeverityLow,
		EscalationLevel: database.EscalationL1, Status: database.StatusPendingReview,
		IncidentDatetime: time.Now().UTC(), LocationID: locB.ID, ReporterID: rep.ID,
	}
	require.NoError(t, db.Create(&incB).Error)

	svc := &correctiveaction.Service{DB: db, Audit: auditlog.Noop{}}
	_, err := svc.Create(context.Background(), incA.ID, correctiveaction.CreateInput{
		Description: "A", ActionType: database.ActionImmediate, Priority: database.PriorityLow,
		AssigneeID: rep.ID, DueDate: time.Now().UTC(),
	}, authctx.Principal{ID: off.ID, Role: off.Role}, "ip")
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), incB.ID, correctiveaction.CreateInput{
		Description: "B", ActionType: database.ActionImmediate, Priority: database.PriorityLow,
		AssigneeID: rep.ID, DueDate: time.Now().UTC(),
	}, authctx.Principal{ID: off2.ID, Role: off2.Role}, "ip")
	require.NoError(t, err)

	items, err := svc.Tracker(authctx.Principal{ID: off.ID, Role: off.Role}, correctiveaction.ListFilter{})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "A", items[0].Description)

	overdue := database.CAStatusOverdue
	filtered, err := svc.Tracker(authctx.Principal{ID: off.ID, Role: off.Role}, correctiveaction.ListFilter{Status: string(overdue)})
	require.NoError(t, err)
	require.Len(t, filtered, 0)
}
