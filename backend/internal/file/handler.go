package file

import (
	"net/http"

	"aegis/internal/incident"
	"aegis/pkg/authctx"
	"aegis/pkg/httputil"
	"aegis/pkg/response"
	"aegis/pkg/storage"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	Service *Service
}

// List godoc
// @Summary      List incident files
// @Tags         files
// @Produce      json
// @Param        id   path      string  true  "incident id"
// @Success      200  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/files [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid id")
		return
	}
	items, err := h.Service.List(r.Context(), id, actor)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, items)
}

// Upload godoc
// @Summary      Upload incident evidence
// @Tags         files
// @Accept       multipart/form-data
// @Produce      json
// @Param        id     path      string  true  "incident id"
// @Param        files  formData  file    true  "files"
// @Success      201    {object}  response.Envelope
// @Failure      422    {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/files [post]
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid id")
		return
	}
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid multipart body")
		return
	}
	headers := r.MultipartForm.File["files"]
	if len(headers) == 0 {
		headers = r.MultipartForm.File["file"]
	}
	uploads := make([]Upload, 0, len(headers))
	for _, fh := range headers {
		src, err := fh.Open()
		if err != nil {
			incident.WriteErr(w, err)
			return
		}
		b, err := ReadLimited(src, storage.MaxBytes)
		_ = src.Close()
		if err != nil {
			incident.WriteErr(w, err)
			return
		}
		uploads = append(uploads, Upload{Name: fh.Filename, Content: b})
	}
	rows, err := h.Service.Upload(r.Context(), id, uploads, actor, httputil.ClientIP(r))
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":           row.ID,
			"originalName": row.OriginalName,
			"storedKey":    row.StoredKey,
			"mimeType":     row.MimeType,
			"sizeBytes":    row.SizeBytes,
			"context":      row.Context,
		})
	}
	_ = response.Success(w, http.StatusCreated, out)
}

// Delete godoc
// @Summary      Delete draft incident file
// @Tags         files
// @Param        id      path  string  true  "incident id"
// @Param        fileId  path  string  true  "file id"
// @Success      204
// @Failure      422  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/files/{fileId} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid id")
		return
	}
	fileID, err := uuid.Parse(chi.URLParam(r, "fileId"))
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid fileId")
		return
	}
	if err := h.Service.Delete(id, fileID, actor); err != nil {
		incident.WriteErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
