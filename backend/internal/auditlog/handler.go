package auditlog

import (
	"bytes"
	"context"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"aegis/pkg/database"
	"aegis/pkg/response"

	"github.com/google/uuid"
)

type ListFilter struct {
	From       *time.Time
	To         *time.Time
	UserID     *uuid.UUID
	EntityType string
	Action     string
	Page       int
	PageSize   int
}

func (r *Repository) List(ctx context.Context, f ListFilter) ([]database.AuditLog, int64, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 50
	}
	q := database.With(ctx, r.DB).Model(&database.AuditLog{})
	if f.From != nil {
		q = q.Where("created_at >= ?", f.From.UTC())
	}
	if f.To != nil {
		q = q.Where("created_at <= ?", f.To.UTC())
	}
	if f.UserID != nil {
		q = q.Where("user_id = ?", *f.UserID)
	}
	if f.EntityType != "" {
		q = q.Where("entity_type = ?", f.EntityType)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []database.AuditLog
	err := q.Order("created_at DESC").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&rows).Error
	return rows, total, err
}

func AuditCSV(rows []database.AuditLog) [][]string {
	out := [][]string{{"createdAt", "userId", "userRole", "ipAddress", "entityType", "entityId", "action"}}
	for _, r := range rows {
		out = append(out, []string{
			r.CreatedAt.UTC().Format(time.RFC3339Nano),
			r.UserID.String(),
			r.UserRole,
			r.IPAddress,
			r.EntityType,
			r.EntityID,
			string(r.Action),
		})
	}
	return out
}

type Handler struct {
	Repo *Repository
}

// List godoc
// @Summary      List audit logs
// @Tags         audit-logs
// @Produce      json
// @Param        format      query     string  false  "json|csv"
// @Param        from        query     string  false  "RFC3339"
// @Param        to          query     string  false  "RFC3339"
// @Param        userId      query     string  false  "user id"
// @Param        entityType  query     string  false  "entity type"
// @Param        action      query     string  false  "action"
// @Success      200         {object}  response.Envelope
// @Failure      401         {object}  response.Envelope
// @Failure      403         {object}  response.Envelope
// @Failure      422         {object}  response.Envelope
// @Security     BearerAuth
// @Router       /audit-logs [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ListFilter{
		EntityType: q.Get("entityType"),
		Action:     q.Get("action"),
	}
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.PageSize, _ = strconv.Atoi(q.Get("pageSize"))
	if v := q.Get("userId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid userId")
			return
		}
		f.UserID = &id
	}
	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid from")
			return
		}
		f.From = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid to")
			return
		}
		f.To = &t
	}
	rows, total, err := h.Repo.List(r.Context(), f)
	if err != nil {
		if response.WriteTimeout(w, err) {
			return
		}
		_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	if q.Get("format") == "csv" {
		table := AuditCSV(rows)
		var buf bytes.Buffer
		cw := csv.NewWriter(&buf)
		if err := cw.WriteAll(table); err != nil {
			_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "csv encode failed")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="audit-logs.csv"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
		return
	}
	_ = response.Success(w, http.StatusOK, map[string]any{
		"items": rows,
		"total": total,
		"page":  f.Page,
	})
}
