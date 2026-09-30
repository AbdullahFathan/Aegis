package workflow_test

import (
	"context"
	"testing"
	"time"

	"aegis/internal/auditlog"
	"aegis/internal/auth"
	correctiveaction "aegis/internal/corrective_action"
	"aegis/internal/incident"
	"aegis/internal/workflow"
	"aegis/pkg/authctx"
	"aegis/pkg/database"

	"github.com/google/uuid"

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

func seedUser(t *testing.T, db *gorm.DB, email string, role database.Role) database.User {
	t.Helper()
	hash, err := auth.HashPassword("password12")
	require.NoError(t, err)
	u := database.User{Email: email, PasswordHash: hash, Name: email, Role: role, Status: database.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	return u
}

func setupSubmitted(t *testing.T, db *gorm.DB) (database.Incident, database.User, database.User, database.User, database.User) {
	t.Helper()
	rep := seedUser(t, db, "rep@example.com", database.RoleReporter)
	sup := seedUser(t, db, "sup@example.com", database.RoleSupervisor)
	off := seedUser(t, db, "off@example.com", database.RoleHSEOfficer)
	mgr := seedUser(t, db, "mgr@example.com", database.RoleHSEManager)
	loc := database.Location{
		Name: "A", Code: "TMB-A", Type: database.LocationTambang,
		SupervisorID: sup.ID, HSEOfficerID: off.ID, IsActive: true,
	}
	require.NoError(t, db.Create(&loc).Error)
	incSvc := &incident.Service{Repo: &incident.Repository{DB: db}, Audit: auditlog.Noop{}}
	actor := authctx.Principal{ID: rep.ID, Role: rep.Role}
	row, err := incSvc.Create(context.Background(), incident.CreateInput{
		Title: "Slip", Description: longDesc, Category: database.CategoryNearMiss,
		Severity: database.SeverityLow, IncidentDatetime: time.Now().UTC(), LocationID: loc.ID,
	}, actor, "ip")
	require.NoError(t, err)
	row, err = incSvc.Submit(context.Background(), row.ID, actor, "ip")
	require.NoError(t, err)
	return row, rep, sup, off, mgr
}

func TestReporterCannotVerify(t *testing.T) {
	db := testDB(t)
	row, rep, _, _, _ := setupSubmitted(t, db)
	wf := &workflow.Service{DB: db, Audit: auditlog.Noop{}}
	_, err := wf.Verify(context.Background(), row.ID, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip", "")
	require.ErrorIs(t, err, incident.ErrForbidden)
}

func TestCloseFromDraftRejected(t *testing.T) {
	db := testDB(t)
	rep := seedUser(t, db, "rep@example.com", database.RoleReporter)
	sup := seedUser(t, db, "sup@example.com", database.RoleSupervisor)
	off := seedUser(t, db, "off@example.com", database.RoleHSEOfficer)
	mgr := seedUser(t, db, "mgr@example.com", database.RoleHSEManager)
	sa := seedUser(t, db, "sa@example.com", database.RoleSuperAdmin)
	loc := database.Location{
		Name: "A", Code: "TMB-A", Type: database.LocationTambang,
		SupervisorID: sup.ID, HSEOfficerID: off.ID, IsActive: true,
	}
	require.NoError(t, db.Create(&loc).Error)
	incSvc := &incident.Service{Repo: &incident.Repository{DB: db}, Audit: auditlog.Noop{}}
	draft, err := incSvc.Create(context.Background(), incident.CreateInput{
		Title: "Slip", Description: longDesc, Category: database.CategoryNearMiss,
		Severity: database.SeverityLow, IncidentDatetime: time.Now().UTC(), LocationID: loc.ID,
	}, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.NoError(t, err)
	wf := &workflow.Service{DB: db, Audit: auditlog.Noop{}}
	_, err = wf.Close(context.Background(), draft.ID, authctx.Principal{ID: sa.ID, Role: sa.Role}, "ip", "")
	require.ErrorIs(t, err, incident.ErrIllegal)
	_ = mgr
}

func TestRejectRequiresCommentAndVerifyCloseTimeline(t *testing.T) {
	db := testDB(t)
	row, _, sup, off, mgr := setupSubmitted(t, db)
	wf := &workflow.Service{DB: db, Audit: &auditlog.Repository{DB: db}}

	_, err := wf.Reject(context.Background(), row.ID, authctx.Principal{ID: sup.ID, Role: sup.Role}, "ip", "")
	require.ErrorIs(t, err, incident.ErrValidation)

	rejected, err := wf.Reject(context.Background(), row.ID, authctx.Principal{ID: sup.ID, Role: sup.Role}, "ip", "incomplete evidence")
	require.NoError(t, err)
	require.Equal(t, database.StatusRejected, rejected.Status)

	incSvc := &incident.Service{Repo: &incident.Repository{DB: db}, Audit: auditlog.Noop{}}
	repActor := authctx.Principal{ID: rejected.ReporterID, Role: database.RoleReporter}
	resubmitted, err := incSvc.Submit(context.Background(), rejected.ID, repActor, "ip")
	require.NoError(t, err)

	verified, err := wf.Verify(context.Background(), resubmitted.ID, authctx.Principal{ID: sup.ID, Role: sup.Role}, "ip", "ok")
	require.NoError(t, err)
	require.Equal(t, database.StatusUnderInvestigation, verified.Status)

	_, err = wf.Verify(context.Background(), verified.ID, authctx.Principal{ID: mgr.ID, Role: mgr.Role}, "ip", "")
	require.ErrorIs(t, err, incident.ErrIllegal)

	closed, err := wf.Close(context.Background(), verified.ID, authctx.Principal{ID: mgr.ID, Role: mgr.Role}, "ip", "done")
	require.NoError(t, err)
	require.Equal(t, database.StatusClosed, closed.Status)
	require.NotNil(t, closed.ClosedAt)

	_, err = wf.Verify(context.Background(), closed.ID, authctx.Principal{ID: sup.ID, Role: sup.Role}, "ip", "")
	require.ErrorIs(t, err, incident.ErrIllegal)

	logs, err := wf.Timeline(context.Background(), closed.ID, authctx.Principal{ID: mgr.ID, Role: mgr.Role})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(logs), 4)
	require.Equal(t, database.StatusDraft, logs[0].FromStatus)
	require.Equal(t, database.StatusPendingReview, logs[0].ToStatus)
	require.False(t, logs[0].CreatedAt.IsZero())
	_ = off
}

func TestStartCorrectiveActionThenClose(t *testing.T) {
	db := testDB(t)
	row, _, sup, off, mgr := setupSubmitted(t, db)
	wf := &workflow.Service{DB: db, Audit: auditlog.Noop{}}
	verified, err := wf.Verify(context.Background(), row.ID, authctx.Principal{ID: sup.ID, Role: sup.Role}, "ip", "")
	require.NoError(t, err)
	ca, err := wf.StartCorrectiveAction(context.Background(), verified.ID, authctx.Principal{ID: off.ID, Role: off.Role}, "ip", "")
	require.NoError(t, err)
	require.Equal(t, database.StatusCorrectiveAction, ca.Status)
	closed, err := wf.Close(context.Background(), ca.ID, authctx.Principal{ID: mgr.ID, Role: mgr.Role}, "ip", "")
	require.NoError(t, err)
	require.Equal(t, database.StatusClosed, closed.Status)
}

func TestCloseRejectedWhenCANotVerified(t *testing.T) {
	db := testDB(t)
	row, rep, sup, off, mgr := setupSubmitted(t, db)
	wf := &workflow.Service{DB: db, Audit: auditlog.Noop{}}
	verified, err := wf.Verify(context.Background(), row.ID, authctx.Principal{ID: sup.ID, Role: sup.Role}, "ip", "")
	require.NoError(t, err)
	caSvc := &correctiveaction.Service{DB: db, Audit: auditlog.Noop{}}
	_, err = caSvc.Create(context.Background(), verified.ID, correctiveaction.CreateInput{
		Description: "fix floor", ActionType: database.ActionImmediate, Priority: database.PriorityHigh,
		AssigneeID: rep.ID, DueDate: time.Now().UTC().Add(24 * time.Hour),
	}, authctx.Principal{ID: off.ID, Role: off.Role}, "ip")
	require.NoError(t, err)
	_, err = wf.Close(context.Background(), verified.ID, authctx.Principal{ID: mgr.ID, Role: mgr.Role}, "ip", "")
	require.ErrorIs(t, err, incident.ErrIllegal)
}

type recEmerg struct{ n int }

func (r *recEmerg) NotifyFatality(context.Context, uuid.UUID, string) error {
	r.n++
	return nil
}

func TestFatalityNotifierOnSubmit(t *testing.T) {
	db := testDB(t)
	rep := seedUser(t, db, "rep@example.com", database.RoleReporter)
	sup := seedUser(t, db, "sup@example.com", database.RoleSupervisor)
	off := seedUser(t, db, "off@example.com", database.RoleHSEOfficer)
	loc := database.Location{
		Name: "A", Code: "TMB-A", Type: database.LocationTambang,
		SupervisorID: sup.ID, HSEOfficerID: off.ID, IsActive: true,
	}
	require.NoError(t, db.Create(&loc).Error)
	rec := &recEmerg{}
	incSvc := &incident.Service{Repo: &incident.Repository{DB: db}, Audit: auditlog.Noop{}, Emergency: rec}
	row, err := incSvc.Create(context.Background(), incident.CreateInput{
		Title: "Fatal", Description: longDesc, Category: database.CategoryFatality,
		Severity: database.SeverityCritical, IncidentDatetime: time.Now().UTC(), LocationID: loc.ID,
	}, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.NoError(t, err)
	_, err = incSvc.Submit(context.Background(), row.ID, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.NoError(t, err)
	require.Equal(t, 1, rec.n)
}
