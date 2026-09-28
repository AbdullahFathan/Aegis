package notification

import (
	"encoding/json"
	"net/http"

	"aegis/internal/incident"
	"aegis/pkg/authctx"
	"aegis/pkg/database"
	"aegis/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	Service *Service
}

type prefBody struct {
	EventType    string `json:"eventType"`
	EmailEnabled bool   `json:"emailEnabled"`
}

// List godoc
// @Summary      List my notifications
// @Tags         notifications
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /notifications [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	items, err := h.Service.List(actor)
	if err != nil {
		_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{
			"id":            it.ID,
			"type":          it.Type,
			"title":         it.Title,
			"body":          it.Body,
			"priority":      it.Priority,
			"isRead":        it.IsRead,
			"referenceType": it.ReferenceType,
			"referenceId":   it.ReferenceID,
			"createdAt":     it.CreatedAt,
		})
	}
	_ = response.Success(w, http.StatusOK, out)
}

// MarkRead godoc
// @Summary      Mark notification read
// @Tags         notifications
// @Param        id   path  string  true  "id"
// @Success      204
// @Security     BearerAuth
// @Router       /notifications/{id}/read [patch]
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
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
	if err := h.Service.MarkRead(id, actor); err != nil {
		incident.WriteErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PutPreference godoc
// @Summary      Set email preference for an event type
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        request  body      prefBody  true  "preference"
// @Success      200      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /notifications/preferences [put]
func (h *Handler) PutPreference(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	var body prefBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.EventType == "" {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	if err := h.Service.SetEmailPreference(actor.ID, database.NotificationType(body.EventType), body.EmailEnabled); err != nil {
		_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	_ = response.Success(w, http.StatusOK, map[string]any{"eventType": body.EventType, "emailEnabled": body.EmailEnabled})
}
