package notifier

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type EmergencyNotifier interface {
	NotifyFatality(ctx context.Context, incidentID uuid.UUID, title string) error
}

type NoopEmergency struct{}

func (NoopEmergency) NotifyFatality(context.Context, uuid.UUID, string) error { return nil }

type LogEmergency struct {
	Log *zap.Logger
}

func (n LogEmergency) NotifyFatality(ctx context.Context, incidentID uuid.UUID, title string) error {
	if n.Log == nil {
		return nil
	}
	n.Log.Info("emergency_fatality",
		zap.String("incidentId", incidentID.String()),
		zap.String("title", title),
	)
	return nil
}

type Mailer interface {
	Send(ctx context.Context, toEmail, subject, body string) error
}

type NoopMailer struct{}

func (NoopMailer) Send(context.Context, string, string, string) error { return nil }
