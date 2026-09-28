package notification_test

import (
	"context"
	"testing"
	"time"

	"aegis/internal/auth"
	"aegis/internal/notification"
	"aegis/pkg/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type recMailer struct {
	n int
}

func (m *recMailer) Send(context.Context, string, string, string) error {
	m.n++
	return nil
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	auth.SetBcryptCost(bcrypt.MinCost)
	db, err := database.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(db))
	return db
}

func seedLoc(t *testing.T, db *gorm.DB) (database.Incident, database.Location, database.User, database.User, database.User) {
	t.Helper()
	hash, err := auth.HashPassword("password12")
	require.NoError(t, err)
	rep := database.User{Email: "rep@example.com", PasswordHash: hash, Name: "r", Role: database.RoleReporter, Status: database.UserStatusActive}
	sup := database.User{Email: "sup@example.com", PasswordHash: hash, Name: "s", Role: database.RoleSupervisor, Status: database.UserStatusActive}
	off := database.User{Email: "off@example.com", PasswordHash: hash, Name: "o", Role: database.RoleHSEOfficer, Status: database.UserStatusActive}
	mgr := database.User{Email: "mgr@example.com", PasswordHash: hash, Name: "m", Role: database.RoleHSEManager, Status: database.UserStatusActive}
	require.NoError(t, db.Create(&rep).Error)
	require.NoError(t, db.Create(&sup).Error)
	require.NoError(t, db.Create(&off).Error)
	require.NoError(t, db.Create(&mgr).Error)
	loc := database.Location{Name: "A", Code: "TMB-A", Type: database.LocationTambang, SupervisorID: sup.ID, HSEOfficerID: off.ID, IsActive: true}
	require.NoError(t, db.Create(&loc).Error)
	inc := database.Incident{
		Title: "Slip", Description: "x", Category: database.CategoryNearMiss, Severity: database.SeverityLow,
		EscalationLevel: database.EscalationL1, Status: database.StatusPendingReview,
		IncidentDatetime: time.Now().UTC(), LocationID: loc.ID, ReporterID: rep.ID,
	}
	require.NoError(t, db.Create(&inc).Error)
	return inc, loc, rep, sup, mgr
}

func TestRecipientsTableDriven(t *testing.T) {
	db := testDB(t)
	inc, loc, rep, sup, mgr := seedLoc(t, db)
	svc := &notification.Service{DB: db}

	svc.OnIncidentSubmitted(context.Background(), inc, loc)
	assertRecipients(t, db, database.NotifIncidentSubmitted, sup.ID)

	inc.Category = database.CategoryFatality
	svc.OnIncidentSubmitted(context.Background(), inc, loc)
	var crit []database.Notification
	require.NoError(t, db.Where("type = ?", database.NotifIncidentEscalated).Find(&crit).Error)
	ids := map[uuid.UUID]struct{}{}
	for _, n := range crit {
		ids[n.RecipientID] = struct{}{}
	}
	require.Contains(t, ids, loc.SupervisorID)
	require.Contains(t, ids, loc.HSEOfficerID)
	require.Contains(t, ids, mgr.ID)

	svc.OnRejected(context.Background(), inc, loc)
	assertRecipients(t, db, database.NotifIncidentRejected, rep.ID)

	ca := database.CorrectiveAction{IncidentID: inc.ID, Description: "fix", ActionType: database.ActionImmediate, Priority: database.PriorityLow, Status: database.CAStatusOpen, AssigneeID: rep.ID, DueDate: time.Now().UTC()}
	require.NoError(t, db.Create(&ca).Error)
	svc.OnCAAssigned(context.Background(), ca, inc, loc)
	assertRecipients(t, db, database.NotifCAAssigned, rep.ID)
}

func assertRecipients(t *testing.T, db *gorm.DB, typ database.NotificationType, want uuid.UUID) {
	t.Helper()
	var rows []database.Notification
	require.NoError(t, db.Where("type = ?", typ).Find(&rows).Error)
	found := false
	for _, r := range rows {
		if r.RecipientID == want {
			found = true
		}
	}
	require.True(t, found, "missing recipient for %s", typ)
}

func TestEmailOptIn(t *testing.T) {
	db := testDB(t)
	inc, loc, _, sup, _ := seedLoc(t, db)
	mail := &recMailer{}
	svc := &notification.Service{DB: db, Mailer: mail}

	svc.OnIncidentSubmitted(context.Background(), inc, loc)
	require.Equal(t, 0, mail.n)

	require.NoError(t, svc.SetEmailPreference(sup.ID, database.NotifIncidentSubmitted, true))
	svc.OnIncidentSubmitted(context.Background(), inc, loc)
	require.Equal(t, 1, mail.n)
}

func TestSLARemindersClock(t *testing.T) {
	db := testDB(t)
	inc, loc, _, sup, _ := seedLoc(t, db)
	_ = loc
	base := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	require.NoError(t, db.Model(&inc).Update("pending_review_at", base).Error)
	once := &notification.MemoryOnce{Now: func() time.Time { return base.Add(21 * time.Hour) }}
	svc := &notification.Service{DB: db, Once: once, Now: func() time.Time { return base.Add(21 * time.Hour) }}

	require.NoError(t, svc.RunSLAReminders(context.Background(), base.Add(21*time.Hour)))
	var warn []database.Notification
	require.NoError(t, db.Where("type = ?", database.NotifSLAWarning).Find(&warn).Error)
	require.Len(t, warn, 1)
	require.Equal(t, sup.ID, warn[0].RecipientID)

	require.NoError(t, svc.RunSLAReminders(context.Background(), base.Add(21*time.Hour)))
	require.NoError(t, db.Where("type = ?", database.NotifSLAWarning).Find(&warn).Error)
	require.Len(t, warn, 1)

	once.Now = func() time.Time { return base.Add(25 * time.Hour) }
	require.NoError(t, svc.RunSLAReminders(context.Background(), base.Add(25*time.Hour)))
	var over []database.Notification
	require.NoError(t, db.Where("type = ?", database.NotifSLAOverdue).Find(&over).Error)
	require.GreaterOrEqual(t, len(over), 1)
}
