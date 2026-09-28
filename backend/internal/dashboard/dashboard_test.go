package dashboard_test

import (
	"testing"
	"time"

	"aegis/internal/auth"
	"aegis/internal/dashboard"
	"aegis/pkg/database"

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

func seedLoc(t *testing.T, db *gorm.DB, code string, sup, off database.User) database.Location {
	t.Helper()
	loc := database.Location{
		Name: code, Code: code, Type: database.LocationTambang,
		SupervisorID: sup.ID, HSEOfficerID: off.ID, IsActive: true,
	}
	require.NoError(t, db.Create(&loc).Error)
	return loc
}

func seedInc(t *testing.T, db *gorm.DB, loc database.Location, reporter database.User, cat database.IncidentCategory, status database.IncidentStatus, when time.Time) database.Incident {
	t.Helper()
	row := database.Incident{
		Title: "Slip", Description: "Worker slipped on wet surface near loading area during morning shift.",
		Category: cat, Severity: database.SeverityMedium, EscalationLevel: database.EscalationL2,
		Status: status, IncidentDatetime: when, LocationID: loc.ID, ReporterID: reporter.ID, HasVictim: false,
	}
	require.NoError(t, db.Create(&row).Error)
	return row
}

func TestSummaryMoMAndPipeline(t *testing.T) {
	db := testDB(t)
	rep := seedUser(t, db, "r@example.com", database.RoleReporter)
	sup := seedUser(t, db, "s@example.com", database.RoleSupervisor)
	off := seedUser(t, db, "o@example.com", database.RoleHSEOfficer)
	loc := seedLoc(t, db, "TMB-A", sup, off)
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	seedInc(t, db, loc, rep, database.CategoryNearMiss, database.StatusPendingReview, time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC))
	seedInc(t, db, loc, rep, database.CategoryLTI, database.StatusClosed, time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC))
	seedInc(t, db, loc, rep, database.CategoryNearMiss, database.StatusDraft, time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC))
	seedInc(t, db, loc, rep, database.CategoryFirstAid, database.StatusClosed, time.Date(2026, 8, 10, 8, 0, 0, 0, time.UTC))
	ca := database.CorrectiveAction{
		IncidentID:  seedInc(t, db, loc, rep, database.CategoryNearMiss, database.StatusCorrectiveAction, time.Date(2026, 9, 8, 8, 0, 0, 0, time.UTC)).ID,
		Description: "Fix floor", ActionType: database.ActionImmediate, Priority: database.PriorityHigh,
		Status: database.CAStatusOverdue, AssigneeID: off.ID, DueDate: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, db.Create(&ca).Error)

	svc := &dashboard.Service{DB: db, Now: func() time.Time { return now }}
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := now
	sum, err := svc.Summary(dashboard.Filter{From: &from, To: &to})
	require.NoError(t, err)
	require.Equal(t, int64(3), sum.ThisMonthCount)
	require.Equal(t, int64(1), sum.LastMonthCount)
	require.Equal(t, "up", sum.Trend)
	require.Equal(t, int64(1), sum.CAOverdue)
	require.Equal(t, int64(1), sum.Pipeline.PendingReview)
	require.Equal(t, int64(1), sum.Pipeline.Closed)
	require.Equal(t, int64(1), sum.Pipeline.CorrectiveAction)
}

func TestTrendsAndHeatmapFilterSite(t *testing.T) {
	db := testDB(t)
	rep := seedUser(t, db, "r@example.com", database.RoleReporter)
	sup := seedUser(t, db, "s@example.com", database.RoleSupervisor)
	off := seedUser(t, db, "o@example.com", database.RoleHSEOfficer)
	supB := seedUser(t, db, "sb@example.com", database.RoleSupervisor)
	offB := seedUser(t, db, "ob@example.com", database.RoleHSEOfficer)
	locA := seedLoc(t, db, "TMB-A", sup, off)
	locB := seedLoc(t, db, "TMB-B", supB, offB)
	now := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	seedInc(t, db, locA, rep, database.CategoryLTI, database.StatusClosed, time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC))
	seedInc(t, db, locA, rep, database.CategoryNearMiss, database.StatusPendingReview, time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC))
	seedInc(t, db, locB, rep, database.CategoryLTI, database.StatusClosed, time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC))

	svc := &dashboard.Service{DB: db, Now: func() time.Time { return now }}
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := now
	f := dashboard.Filter{From: &from, To: &to, LocationID: &locA.ID}
	tr, err := svc.Trends(f)
	require.NoError(t, err)
	require.Len(t, tr, 12)
	var sep int64
	for _, b := range tr {
		if b.Year == 2026 && b.Month == 9 {
			sep = b.Count
		}
	}
	require.Equal(t, int64(2), sep)

	heat, err := svc.Heatmap(dashboard.Filter{From: &from, To: &to})
	require.NoError(t, err)
	require.Len(t, heat, 3)
	codes := map[string]int64{}
	for _, c := range heat {
		codes[c.LocationCode+":"+c.Category] = c.Count
	}
	require.Equal(t, int64(1), codes["TMB-A:LTI"])
	require.Equal(t, int64(1), codes["TMB-A:NEAR_MISS"])
	require.Equal(t, int64(1), codes["TMB-B:LTI"])

	heatA, err := svc.Heatmap(f)
	require.NoError(t, err)
	for _, c := range heatA {
		require.Equal(t, locA.ID.String(), c.LocationID)
	}
}

func BenchmarkDashboardSummary(b *testing.B) {
	auth.SetBcryptCost(bcrypt.MinCost)
	db, err := database.OpenSQLite("file:bench-summary?mode=memory&cache=shared")
	if err != nil {
		b.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		b.Fatal(err)
	}
	sup := database.User{Email: "s@example.com", PasswordHash: "x", Name: "s", Role: database.RoleSupervisor, Status: database.UserStatusActive}
	off := database.User{Email: "o@example.com", PasswordHash: "x", Name: "o", Role: database.RoleHSEOfficer, Status: database.UserStatusActive}
	rep := database.User{Email: "r@example.com", PasswordHash: "x", Name: "r", Role: database.RoleReporter, Status: database.UserStatusActive}
	if err := db.Create(&sup).Error; err != nil {
		b.Fatal(err)
	}
	if err := db.Create(&off).Error; err != nil {
		b.Fatal(err)
	}
	if err := db.Create(&rep).Error; err != nil {
		b.Fatal(err)
	}
	loc := database.Location{Name: "A", Code: "TMB-A", Type: database.LocationTambang, SupervisorID: sup.ID, HSEOfficerID: off.ID, IsActive: true}
	if err := db.Create(&loc).Error; err != nil {
		b.Fatal(err)
	}
	when := time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC)
	for i := 0; i < 50; i++ {
		row := database.Incident{
			Title: "Slip", Description: "Worker slipped on wet surface near loading area during morning shift.",
			Category: database.CategoryNearMiss, Severity: database.SeverityLow, EscalationLevel: database.EscalationL1,
			Status: database.StatusPendingReview, IncidentDatetime: when, LocationID: loc.ID, ReporterID: rep.ID,
		}
		if err := db.Create(&row).Error; err != nil {
			b.Fatal(err)
		}
	}
	svc := &dashboard.Service{DB: db, Now: func() time.Time { return time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC) }}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.Summary(dashboard.Filter{}); err != nil {
			b.Fatal(err)
		}
	}
}
