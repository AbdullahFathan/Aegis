package auditlog

import (
	"context"
	"testing"

	"aegis/pkg/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestInsertCreatedAndUpdated(t *testing.T) {
	db, err := database.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(db))

	actor := database.User{
		Email:        "admin@example.com",
		PasswordHash: "x",
		Name:         "Admin",
		Role:         database.RoleAdmin,
		Status:       database.UserStatusActive,
	}
	require.NoError(t, db.Create(&actor).Error)

	repo := &Repository{DB: db}
	ctx := context.Background()
	entityID := uuid.New().String()

	require.NoError(t, repo.Insert(ctx, Entry{
		UserID:     actor.ID,
		UserRole:   string(actor.Role),
		IPAddress:  "127.0.0.1",
		EntityType: "User",
		EntityID:   entityID,
		Action:     database.AuditCreated,
		After:      map[string]any{"email": "n@example.com"},
	}))
	require.NoError(t, repo.Insert(ctx, Entry{
		UserID:     actor.ID,
		UserRole:   string(actor.Role),
		IPAddress:  "127.0.0.1",
		EntityType: "User",
		EntityID:   entityID,
		Action:     database.AuditUpdated,
		Before:     map[string]any{"role": "REPORTER"},
		After:      map[string]any{"role": "SUPERVISOR"},
	}))

	var rows []database.AuditLog
	require.NoError(t, db.Order("created_at ASC").Find(&rows).Error)
	require.Len(t, rows, 2)
	require.Equal(t, database.AuditCreated, rows[0].Action)
	require.Equal(t, database.AuditUpdated, rows[1].Action)
	require.Equal(t, entityID, rows[0].EntityID)
}

func TestUpdateAndDeleteAreRejected(t *testing.T) {
	repo := &Repository{}
	err := repo.Update(context.Background(), uuid.New())
	require.Error(t, err)
	require.Contains(t, err.Error(), "immutable")
	err = repo.Delete(context.Background(), uuid.New())
	require.Error(t, err)
	require.Contains(t, err.Error(), "immutable")
}
