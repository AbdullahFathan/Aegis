package database

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const afterCommitTimeout = 2 * time.Second

// With binds ctx to db so GORM cancels the statement when the context ends.
func With(ctx context.Context, db *gorm.DB) *gorm.DB {
	if db == nil || ctx == nil {
		return db
	}
	return db.WithContext(ctx)
}

// AfterCommit returns a short context that survives client disconnect and still
// cannot hang. Use it for audit and notification writes that follow a commit.
func AfterCommit(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(ctx), afterCommitTimeout)
}

// IsTimeout reports a canceled context or a Postgres statement cancel (57014).
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "57014" {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "context canceled") ||
		strings.Contains(msg, "canceling statement due to user request")
}
