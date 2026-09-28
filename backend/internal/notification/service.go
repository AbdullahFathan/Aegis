package notification

import (
	"context"
	"errors"
	"strings"
	"time"

	"aegis/internal/incident"
	"aegis/pkg/authctx"
	"aegis/pkg/database"
	"aegis/pkg/notifier"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	DB     *gorm.DB
	Mailer notifier.Mailer
	Log    *zap.Logger
	Once   Once
	Now    func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s *Service) mailer() notifier.Mailer {
	if s.Mailer != nil {
		return s.Mailer
	}
	return notifier.NoopMailer{}
}

type Event struct {
	Type          database.NotificationType
	Priority      database.NotificationPriority
	Title         string
	Body          string
	ReferenceType string
	ReferenceID   uuid.UUID
	RecipientIDs  []uuid.UUID
}

func (s *Service) Dispatch(ctx context.Context, e Event) {
	seen := map[uuid.UUID]struct{}{}
	for _, id := range e.RecipientIDs {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		row := database.Notification{
			RecipientID:   id,
			Type:          e.Type,
			Title:         e.Title,
			Body:          e.Body,
			Priority:      e.Priority,
			IsRead:        false,
			ReferenceType: e.ReferenceType,
			ReferenceID:   e.ReferenceID,
		}
		if err := s.DB.Create(&row).Error; err != nil {
			if s.Log != nil {
				s.Log.Error("notification_insert_failed", zap.Error(err))
			}
			continue
		}
		s.maybeEmail(ctx, id, e)
	}
}

func (s *Service) maybeEmail(ctx context.Context, userID uuid.UUID, e Event) {
	var pref database.NotificationPreference
	err := s.DB.Where("user_id = ? AND event_type = ?", userID, e.Type).First(&pref).Error
	if err != nil || !pref.EmailEnabled {
		return
	}
	var user database.User
	if err := s.DB.First(&user, "id = ?", userID).Error; err != nil {
		return
	}
	if err := s.mailer().Send(ctx, user.Email, e.Title, e.Body); err != nil && s.Log != nil {
		s.Log.Error("notification_email_failed", zap.Error(err))
	}
}

func (s *Service) SetEmailPreference(userID uuid.UUID, event database.NotificationType, enabled bool) error {
	var pref database.NotificationPreference
	err := s.DB.Where("user_id = ? AND event_type = ?", userID, event).First(&pref).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		pref = database.NotificationPreference{UserID: userID, EventType: event, EmailEnabled: enabled}
		return s.DB.Create(&pref).Error
	}
	if err != nil {
		return err
	}
	pref.EmailEnabled = enabled
	return s.DB.Save(&pref).Error
}

func (s *Service) List(actor authctx.Principal) ([]database.Notification, error) {
	var items []database.Notification
	err := s.DB.Where("recipient_id = ?", actor.ID).Order("created_at DESC").Find(&items).Error
	return items, err
}

func (s *Service) MarkRead(id uuid.UUID, actor authctx.Principal) error {
	var row database.Notification
	err := s.DB.First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return incident.ErrNotFound
	}
	if row.RecipientID != actor.ID {
		return incident.ErrForbidden
	}
	row.IsRead = true
	return s.DB.Save(&row).Error
}

func (s *Service) managerIDs() []uuid.UUID {
	var users []database.User
	_ = s.DB.Where("role = ? AND status = ?", database.RoleHSEManager, database.UserStatusActive).Find(&users).Error
	ids := make([]uuid.UUID, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	return ids
}

func (s *Service) OnIncidentSubmitted(ctx context.Context, inc database.Incident, loc database.Location) {
	critical := inc.Category == database.CategoryLTI || inc.Category == database.CategoryFatality
	title := "Laporan insiden baru"
	if inc.IncidentNumber != nil {
		title = "Laporan " + *inc.IncidentNumber
	}
	if critical {
		ids := []uuid.UUID{loc.SupervisorID, loc.HSEOfficerID}
		ids = append(ids, s.managerIDs()...)
		s.Dispatch(ctx, Event{
			Type: database.NotifIncidentEscalated, Priority: database.NotifCritical,
			Title: title, Body: "Insiden LTI/Fatality memerlukan eskalasi.",
			ReferenceType: "incident", ReferenceID: inc.ID, RecipientIDs: ids,
		})
		return
	}
	s.Dispatch(ctx, Event{
		Type: database.NotifIncidentSubmitted, Priority: database.NotifHigh,
		Title: title, Body: "Laporan baru menunggu review supervisor.",
		ReferenceType: "incident", ReferenceID: inc.ID, RecipientIDs: []uuid.UUID{loc.SupervisorID},
	})
}

func (s *Service) OnVerified(ctx context.Context, inc database.Incident, loc database.Location) {
	s.Dispatch(ctx, Event{
		Type: database.NotifIncidentVerified, Priority: database.NotifMedium,
		Title: "Laporan diverifikasi", Body: "Supervisor memverifikasi laporan.",
		ReferenceType: "incident", ReferenceID: inc.ID, RecipientIDs: []uuid.UUID{loc.HSEOfficerID},
	})
}

func (s *Service) OnRejected(ctx context.Context, inc database.Incident, loc database.Location) {
	_ = loc
	s.Dispatch(ctx, Event{
		Type: database.NotifIncidentRejected, Priority: database.NotifMedium,
		Title: "Laporan dikembalikan", Body: "Laporan dikembalikan untuk revisi.",
		ReferenceType: "incident", ReferenceID: inc.ID, RecipientIDs: []uuid.UUID{inc.ReporterID},
	})
}

func (s *Service) OnClosed(ctx context.Context, inc database.Incident, loc database.Location) {
	s.Dispatch(ctx, Event{
		Type: database.NotifIncidentClosed, Priority: database.NotifInfo,
		Title: "Laporan ditutup", Body: "Laporan final telah ditutup.",
		ReferenceType: "incident", ReferenceID: inc.ID,
		RecipientIDs: []uuid.UUID{inc.ReporterID, loc.SupervisorID},
	})
}

func (s *Service) OnCAAssigned(ctx context.Context, ca database.CorrectiveAction, inc database.Incident, loc database.Location) {
	_ = loc
	s.Dispatch(ctx, Event{
		Type: database.NotifCAAssigned, Priority: database.NotifMedium,
		Title: "Corrective action di-assign", Body: strings.TrimSpace(ca.Description),
		ReferenceType: "corrective_action", ReferenceID: ca.ID,
		RecipientIDs: []uuid.UUID{ca.AssigneeID},
	})
	_ = inc
}

func (s *Service) OnCAOverdue(ctx context.Context, ca database.CorrectiveAction, loc database.Location) {
	ids := []uuid.UUID{ca.AssigneeID, loc.HSEOfficerID}
	ids = append(ids, s.managerIDs()...)
	s.Dispatch(ctx, Event{
		Type: database.NotifCAOverdue, Priority: database.NotifHigh,
		Title: "Corrective action overdue", Body: "CA melewati due date.",
		ReferenceType: "corrective_action", ReferenceID: ca.ID, RecipientIDs: ids,
	})
}

func (s *Service) OnCADueSoon(ctx context.Context, ca database.CorrectiveAction, loc database.Location) {
	s.Dispatch(ctx, Event{
		Type: database.NotifCADueSoon, Priority: database.NotifMedium,
		Title: "Corrective action mendekati due date", Body: "CA jatuh tempo dalam 2 hari.",
		ReferenceType: "corrective_action", ReferenceID: ca.ID,
		RecipientIDs: []uuid.UUID{ca.AssigneeID, loc.HSEOfficerID},
	})
}

func (s *Service) claim(ctx context.Context, key string, ttl time.Duration) bool {
	if s.Once == nil {
		return true
	}
	return s.Once.Claim(ctx, key, ttl)
}

func (s *Service) RunSLAReminders(ctx context.Context, now time.Time) error {
	var items []database.Incident
	err := s.DB.Where("status = ? AND pending_review_at IS NOT NULL", database.StatusPendingReview).Find(&items).Error
	if err != nil {
		return err
	}
	for _, inc := range items {
		if inc.PendingReviewAt == nil {
			continue
		}
		elapsed := now.Sub(*inc.PendingReviewAt)
		var loc database.Location
		if err := s.DB.First(&loc, "id = ?", inc.LocationID).Error; err != nil {
			continue
		}
		if elapsed >= 24*time.Hour {
			key := "sla-overdue:" + inc.ID.String()
			if !s.claim(ctx, key, 24*time.Hour) {
				continue
			}
			s.Dispatch(ctx, Event{
				Type: database.NotifSLAOverdue, Priority: database.NotifHigh,
				Title: "SLA supervisor terlewat", Body: "Review supervisor melewati 24 jam.",
				ReferenceType: "incident", ReferenceID: inc.ID,
				RecipientIDs: []uuid.UUID{loc.SupervisorID, loc.HSEOfficerID},
			})
			continue
		}
		if elapsed >= 20*time.Hour {
			key := "sla-warn:" + inc.ID.String()
			if !s.claim(ctx, key, 24*time.Hour) {
				continue
			}
			s.Dispatch(ctx, Event{
				Type: database.NotifSLAWarning, Priority: database.NotifHigh,
				Title: "SLA supervisor hampir habis", Body: "Sisa waktu review kurang dari 4 jam.",
				ReferenceType: "incident", ReferenceID: inc.ID,
				RecipientIDs: []uuid.UUID{loc.SupervisorID},
			})
		}
	}
	return nil
}
