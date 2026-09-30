package file_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"aegis/internal/auditlog"
	"aegis/internal/file"
	"aegis/internal/incident"
	"aegis/pkg/database"

	"github.com/stretchr/testify/require"
)

func TestSecurityFakeMIMERejected(t *testing.T) {
	db := testDB(t)
	row, actor, fs, _ := seed(t, db)
	_, err := fs.Upload(context.Background(), row.ID, []file.Upload{{
		Name: "photo.jpg", Content: []byte("%PDF-1.4 not a jpeg"),
	}}, actor, "ip", nil)
	require.ErrorIs(t, err, incident.ErrValidation)
}

func TestSecuritySignedURLExpiry(t *testing.T) {
	db := testDB(t)
	row, actor, fs, _ := seed(t, db)
	created, err := fs.Upload(context.Background(), row.ID, []file.Upload{{
		Name: "photo.jpg", Content: jpegBytes(),
	}}, actor, "ip", nil)
	require.NoError(t, err)
	listed, err := fs.List(context.Background(), row.ID, actor)
	require.NoError(t, err)
	url, _ := listed[0]["url"].(string)
	exp := time.Date(2026, 9, 28, 13, 0, 0, 0, time.UTC).Unix()
	require.Contains(t, url, "exp="+strconv.FormatInt(exp, 10))
	require.NotEmpty(t, created[0].StoredKey)
}

func TestSecurityDeleteAfterSubmitRejected(t *testing.T) {
	db := testDB(t)
	row, actor, fs, _ := seed(t, db)
	created, err := fs.Upload(context.Background(), row.ID, []file.Upload{{
		Name: "photo.jpg", Content: jpegBytes(),
	}}, actor, "ip", nil)
	require.NoError(t, err)
	incSvc := &incident.Service{Repo: &incident.Repository{DB: db}, Audit: auditlog.Noop{}}
	_, err = incSvc.Submit(context.Background(), row.ID, actor, "ip")
	require.NoError(t, err)
	err = fs.Delete(context.Background(), row.ID, created[0].ID, actor)
	require.ErrorIs(t, err, incident.ErrIllegal)
	var still database.IncidentFile
	require.NoError(t, db.First(&still, "id = ?", created[0].ID).Error)
}
