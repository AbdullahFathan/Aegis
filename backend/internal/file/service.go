package file

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"aegis/internal/auditlog"
	"aegis/internal/incident"
	"aegis/internal/incscope"
	"aegis/pkg/authctx"
	"aegis/pkg/database"
	"aegis/pkg/storage"

	"github.com/google/uuid"
)

type Clock func() time.Time

type Service struct {
	Repo    *incident.Repository
	Store   storage.ObjectStore
	Audit   auditlog.Writer
	Now     Clock
	Presign time.Duration
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s *Service) presignTTL() time.Duration {
	if s.Presign > 0 {
		return s.Presign
	}
	return storage.SignedURLTTL
}

type Upload struct {
	Name    string
	Content []byte
}

func (s *Service) Upload(ctx context.Context, incidentID uuid.UUID, files []Upload, actor authctx.Principal, ip string, caID *uuid.UUID) ([]database.IncidentFile, error) {
	if len(files) == 0 {
		return nil, incident.WrapValidation("at least one file is required")
	}
	if len(files) > storage.MaxFilesPerRequest {
		return nil, incident.WrapValidation("maximum 5 files per request")
	}
	inc, loc, err := s.loadVisible(ctx, incidentID, actor)
	if err != nil {
		return nil, err
	}
	if !s.canUpload(ctx, actor, inc, loc, caID) {
		return nil, incident.ErrForbidden
	}
	var caPtr *uuid.UUID
	ctxType := database.FileIncidentEvidence
	if caID != nil {
		var ca database.CorrectiveAction
		if err := database.With(ctx, s.Repo.DB).First(&ca, "id = ?", *caID).Error; err != nil || ca.IncidentID != incidentID {
			return nil, incident.WrapValidation("correctiveActionId does not belong to incident")
		}
		caPtr = caID
		ctxType = database.FileCACompletion
	}

	out := make([]database.IncidentFile, 0, len(files))
	for _, f := range files {
		if int64(len(f.Content)) > storage.MaxBytes {
			return nil, incident.WrapValidation("file exceeds 10MB")
		}
		ext := strings.ToLower(filepath.Ext(f.Name))
		want, ok := expectedMIME(ext)
		if !ok {
			return nil, incident.WrapValidation("unsupported file type")
		}
		head := f.Content
		if len(head) > 512 {
			head = head[:512]
		}
		got := sniffMIME(head)
		if got != want {
			return nil, incident.WrapValidation("MIME type does not match file content")
		}
		key := fmt.Sprintf("incidents/%s/%s", incidentID.String(), uuid.New().String())
		if err := s.Store.Put(ctx, key, bytes.NewReader(f.Content), int64(len(f.Content)), want); err != nil {
			return nil, err
		}
		row := database.IncidentFile{
			IncidentID:         incidentID,
			CorrectiveActionID: caPtr,
			UploadedByID:       actor.ID,
			OriginalName:       f.Name,
			StoredKey:          key,
			MimeType:           want,
			SizeBytes:          int64(len(f.Content)),
			Context:            ctxType,
		}
		if err := s.Repo.CreateFile(ctx, &row); err != nil {
			return nil, err
		}
		actx, cancel := database.AfterCommit(ctx)
		_ = s.Audit.Insert(actx, auditlog.Entry{
			UserID: actor.ID, UserRole: string(actor.Role), IPAddress: ip,
			EntityType: "IncidentFile", EntityID: row.ID.String(),
			Action: database.AuditFileUploaded,
			After: map[string]any{
				"incidentId": incidentID.String(),
				"storedKey":  key,
				"mimeType":   want,
				"sizeBytes":  row.SizeBytes,
			},
		})
		cancel()
		out = append(out, row)
	}
	return out, nil
}

func (s *Service) List(ctx context.Context, incidentID uuid.UUID, actor authctx.Principal) ([]map[string]any, error) {
	if _, _, err := s.loadVisible(ctx, incidentID, actor); err != nil {
		return nil, err
	}
	items, err := s.Repo.ListFiles(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	expiry := s.presignTTL()
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		url, err := s.Store.PresignGet(ctx, it.StoredKey, expiry)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id":                 it.ID,
			"originalName":       it.OriginalName,
			"mimeType":           it.MimeType,
			"sizeBytes":          it.SizeBytes,
			"context":            it.Context,
			"correctiveActionId": it.CorrectiveActionID,
			"storedKey":          it.StoredKey,
			"url":                url,
			"expiresIn":          int(expiry.Seconds()),
			"createdAt":          it.CreatedAt,
		})
	}
	return out, nil
}

func (s *Service) Delete(ctx context.Context, incidentID, fileID uuid.UUID, actor authctx.Principal) error {
	inc, loc, err := s.loadVisible(ctx, incidentID, actor)
	if err != nil {
		return err
	}
	if inc.Status != database.StatusDraft {
		return incident.WrapIllegal("files cannot be deleted after submit")
	}
	if inc.ReporterID != actor.ID && actor.Role != database.RoleSuperAdmin && actor.Role != database.RoleAdmin {
		return incident.ErrForbidden
	}
	_ = loc
	f, err := s.Repo.FindFile(ctx, fileID)
	if err != nil {
		return err
	}
	if f.IncidentID != incidentID {
		return incident.ErrNotFound
	}
	return s.Repo.DeleteFile(ctx, fileID)
}

func (s *Service) canUpload(ctx context.Context, actor authctx.Principal, inc database.Incident, loc database.Location, caID *uuid.UUID) bool {
	if inc.Status == database.StatusClosed {
		return false
	}
	if caID != nil {
		var ca database.CorrectiveAction
		if err := database.With(ctx, s.Repo.DB).First(&ca, "id = ?", *caID).Error; err == nil && ca.IncidentID == inc.ID && ca.AssigneeID == actor.ID {
			return true
		}
	}
	return s.canWrite(actor, inc, loc)
}

func (s *Service) canWrite(actor authctx.Principal, inc database.Incident, loc database.Location) bool {
	if inc.Status == database.StatusClosed {
		return false
	}
	if inc.Status == database.StatusDraft || inc.Status == database.StatusRejected {
		return inc.ReporterID == actor.ID
	}
	if actor.Role == database.RoleHSEManager || actor.Role == database.RoleSuperAdmin {
		return true
	}
	return actor.Role == database.RoleHSEOfficer && incscope.IsLocationOfficer(actor, loc)
}

func (s *Service) loadVisible(ctx context.Context, id uuid.UUID, actor authctx.Principal) (database.Incident, database.Location, error) {
	inc, err := s.Repo.Find(ctx, id)
	if err != nil {
		return database.Incident{}, database.Location{}, err
	}
	loc, err := s.Repo.FindLocation(ctx, inc.LocationID)
	if err != nil {
		return database.Incident{}, database.Location{}, err
	}
	if !incscope.CanSee(actor, inc, loc) {
		return database.Incident{}, database.Location{}, incident.ErrNotFound
	}
	return inc, loc, nil
}

func ReadLimited(r io.Reader, max int64) ([]byte, error) {
	var buf bytes.Buffer
	n, err := io.Copy(&buf, io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if n > max {
		return nil, incident.WrapValidation("file exceeds 10MB")
	}
	return buf.Bytes(), nil
}
