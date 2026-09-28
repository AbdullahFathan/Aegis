package report_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"aegis/internal/auth"
	"aegis/internal/incident"
	"aegis/internal/notification"
	"aegis/internal/report"
	"aegis/pkg/authctx"
	"aegis/pkg/database"
	aegispdf "aegis/pkg/pdf"
	"aegis/pkg/storage"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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
	u := database.User{Email: email, PasswordHash: "x", Name: email, Role: role, Status: database.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	return u
}

func TestFrequencyRateFixture(t *testing.T) {
	v, err := report.FrequencyRate(2, 1_000_000)
	require.NoError(t, err)
	require.Equal(t, 2.0, v)
	_, err = report.FrequencyRate(1, 0)
	require.ErrorIs(t, err, incident.ErrValidation)
}

func TestLTIFRUsesWorkHoursAndCSVShape(t *testing.T) {
	db := testDB(t)
	mgr := seedUser(t, db, "m@example.com", database.RoleHSEManager)
	rep := seedUser(t, db, "r@example.com", database.RoleReporter)
	sup := seedUser(t, db, "s@example.com", database.RoleSupervisor)
	off := seedUser(t, db, "o@example.com", database.RoleHSEOfficer)
	loc := database.Location{
		Name: "A", Code: "TMB-A", Type: database.LocationTambang,
		SupervisorID: sup.ID, HSEOfficerID: off.ID, IsActive: true,
	}
	require.NoError(t, db.Create(&loc).Error)
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC)
	svc := &report.Service{DB: db, Queue: &report.MemoryQueue{}, Store: storage.NewMemory(), PDF: aegispdf.Static{Bytes: []byte("%PDF-ok")}}
	_, err := svc.UpsertWorkHours(nil, from, to, 1_000_000)
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Create(&database.Incident{
			Title: "LTI", Description: "Worker slipped on wet surface near loading area during morning shift.",
			Category: database.CategoryLTI, Severity: database.SeverityHigh, EscalationLevel: database.EscalationL3,
			Status: database.StatusClosed, IncidentDatetime: time.Date(2026, 9, 10+i, 8, 0, 0, 0, time.UTC),
			LocationID: loc.ID, ReporterID: rep.ID, HasVictim: true,
		}).Error)
	}
	actor := authctx.Principal{ID: mgr.ID, Role: mgr.Role}
	out, err := svc.LTIFR(actor, report.QueryFilter{From: &from, To: &to})
	require.NoError(t, err)
	require.Equal(t, int64(2), out.LTICount)
	require.Equal(t, 2.0, out.LTIFR)

	csvRows := report.LTIFRCSV(out)
	require.Equal(t, []string{"periodFrom", "periodTo", "ltiCount", "recordableCount", "hours", "ltifr", "trifr"}, csvRows[0])
	b, err := report.EncodeCSV([][]string{{"a", `x,y`}, {"b", `say "hi"`}})
	require.NoError(t, err)
	require.Contains(t, string(b), `"x,y"`)

	_, err = svc.LTIFR(actor, report.QueryFilter{
		From: ptrTime(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		To:   ptrTime(time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)),
	})
	require.ErrorIs(t, err, incident.ErrValidation)
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestEnqueueProcessAndFailedPDFLeavesIncident(t *testing.T) {
	db := testDB(t)
	mgr := seedUser(t, db, "m@example.com", database.RoleHSEManager)
	rep := seedUser(t, db, "r@example.com", database.RoleReporter)
	sup := seedUser(t, db, "s@example.com", database.RoleSupervisor)
	off := seedUser(t, db, "o@example.com", database.RoleHSEOfficer)
	loc := database.Location{
		Name: "A", Code: "TMB-A", Type: database.LocationTambang,
		SupervisorID: sup.ID, HSEOfficerID: off.ID, IsActive: true,
	}
	require.NoError(t, db.Create(&loc).Error)
	inc := database.Incident{
		Title: "Slip", Description: "Worker slipped on wet surface near loading area during morning shift.",
		Category: database.CategoryNearMiss, Severity: database.SeverityLow, EscalationLevel: database.EscalationL1,
		Status: database.StatusPendingReview, IncidentDatetime: time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC),
		LocationID: loc.ID, ReporterID: rep.ID,
	}
	require.NoError(t, db.Create(&inc).Error)
	incID := inc.ID

	q := &report.MemoryQueue{}
	store := storage.NewMemory()
	svc := &report.Service{DB: db, Queue: q, Store: store, PDF: aegispdf.Static{Bytes: []byte("%PDF-ok")}}
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	actor := authctx.Principal{ID: mgr.ID, Role: mgr.Role}
	job, err := svc.EnqueuePDF(context.Background(), actor, database.ReportMonthly, report.QueryFilter{From: &from, To: &to}, nil)
	require.NoError(t, err)
	require.Equal(t, database.ReportPending, job.Status)
	require.NoError(t, svc.ProcessPending(context.Background(), 5))
	got, url, err := svc.GetJob(job.ID, actor)
	require.NoError(t, err)
	require.Equal(t, database.ReportDone, got.Status)
	require.NotNil(t, got.StoredKey)
	require.True(t, strings.HasPrefix(url, "memory://"))

	failQ := &report.MemoryQueue{}
	fail := &report.Service{DB: db, Queue: failQ, Store: store, PDF: aegispdf.Static{Err: errors.New("boom")}}
	job2, err := fail.EnqueuePDF(context.Background(), actor, database.ReportMonthly, report.QueryFilter{From: &from, To: &to}, nil)
	require.NoError(t, err)
	require.NoError(t, fail.ProcessPending(context.Background(), 5))
	failed, _, err := fail.GetJob(job2.ID, actor)
	require.NoError(t, err)
	require.Equal(t, database.ReportFailed, failed.Status)
	var still database.Incident
	require.NoError(t, db.First(&still, "id = ?", incID).Error)
	require.Equal(t, incID, still.ID)
}

func TestMonthlyScheduleOnce(t *testing.T) {
	db := testDB(t)
	mgr := seedUser(t, db, "m@example.com", database.RoleHSEManager)
	q := &report.MemoryQueue{}
	once := &notification.MemoryOnce{}
	notif := &notification.Service{DB: db, Once: once}
	now := time.Date(2026, 9, 1, 0, 30, 0, 0, time.UTC)
	svc := &report.Service{
		DB: db, Queue: q, Store: storage.NewMemory(), PDF: aegispdf.Static{Bytes: []byte("%PDF")},
		Notify: notif, Once: once, Now: func() time.Time { return now }, TZ: time.UTC,
	}
	require.NoError(t, svc.RunMonthlySchedule(context.Background(), now))
	require.NoError(t, svc.RunMonthlySchedule(context.Background(), now))
	var jobs []database.GeneratedReport
	require.NoError(t, db.Find(&jobs).Error)
	require.Len(t, jobs, 1)
	require.Equal(t, database.ReportMonthly, jobs[0].Type)
	var notes []database.Notification
	require.NoError(t, db.Find(&notes).Error)
	require.NotEmpty(t, notes)
	require.Equal(t, mgr.ID, notes[0].RecipientID)
}

func TestOfficerCannotSeeOtherSiteMonthly(t *testing.T) {
	db := testDB(t)
	rep := seedUser(t, db, "r@example.com", database.RoleReporter)
	supA := seedUser(t, db, "sa@example.com", database.RoleSupervisor)
	offA := seedUser(t, db, "oa@example.com", database.RoleHSEOfficer)
	supB := seedUser(t, db, "sb@example.com", database.RoleSupervisor)
	offB := seedUser(t, db, "ob@example.com", database.RoleHSEOfficer)
	locA := database.Location{Name: "A", Code: "A", Type: database.LocationTambang, SupervisorID: supA.ID, HSEOfficerID: offA.ID, IsActive: true}
	locB := database.Location{Name: "B", Code: "B", Type: database.LocationTambang, SupervisorID: supB.ID, HSEOfficerID: offB.ID, IsActive: true}
	require.NoError(t, db.Create(&locA).Error)
	require.NoError(t, db.Create(&locB).Error)
	require.NoError(t, db.Create(&database.Incident{
		Title: "A", Description: "Worker slipped on wet surface near loading area during morning shift.",
		Category: database.CategoryNearMiss, Severity: database.SeverityLow, EscalationLevel: database.EscalationL1,
		Status: database.StatusPendingReview, IncidentDatetime: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC),
		LocationID: locA.ID, ReporterID: rep.ID,
	}).Error)
	require.NoError(t, db.Create(&database.Incident{
		Title: "B", Description: "Worker slipped on wet surface near loading area during morning shift.",
		Category: database.CategoryNearMiss, Severity: database.SeverityLow, EscalationLevel: database.EscalationL1,
		Status: database.StatusPendingReview, IncidentDatetime: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC),
		LocationID: locB.ID, ReporterID: rep.ID,
	}).Error)
	svc := &report.Service{DB: db}
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	items, err := svc.MonthlyRows(authctx.Principal{ID: offA.ID, Role: offA.Role}, report.QueryFilter{From: &from, To: &to})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, locA.ID, items[0].LocationID)
}
