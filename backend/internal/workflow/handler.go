package workflow

import (
	"encoding/json"
	"net/http"

	"aegis/internal/incident"
	"aegis/pkg/authctx"
	"aegis/pkg/httputil"
	"aegis/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	Service *Service
}

type commentBody struct {
	Comment string `json:"comment"`
}

func (h *Handler) actorID(w http.ResponseWriter, r *http.Request) (authctx.Principal, uuid.UUID, bool) {
	actor, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return authctx.Principal{}, uuid.Nil, false
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid id")
		return authctx.Principal{}, uuid.Nil, false
	}
	return actor, id, true
}

func decodeComment(r *http.Request) (string, error) {
	if r.ContentLength == 0 {
		return "", nil
	}
	var body commentBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return "", err
	}
	return body.Comment, nil
}

// Verify godoc
// @Summary      Verify incident (supervisor)
// @Tags         workflow
// @Accept       json
// @Produce      json
// @Param        id       path      string       true  "id"
// @Param        request  body      commentBody  false "comment"
// @Success      200      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/verify [post]
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := h.actorID(w, r)
	if !ok {
		return
	}
	comment, _ := decodeComment(r)
	row, err := h.Service.Verify(r.Context(), id, actor, httputil.ClientIP(r), comment)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, incident.View(row))
}

// Reject godoc
// @Summary      Reject incident
// @Tags         workflow
// @Accept       json
// @Produce      json
// @Param        id       path      string       true  "id"
// @Param        request  body      commentBody  true  "comment"
// @Success      200      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/reject [post]
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := h.actorID(w, r)
	if !ok {
		return
	}
	comment, err := decodeComment(r)
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	row, err := h.Service.Reject(r.Context(), id, actor, httputil.ClientIP(r), comment)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, incident.View(row))
}

// Close godoc
// @Summary      Close incident
// @Tags         workflow
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "id"
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/close [post]
func (h *Handler) Close(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := h.actorID(w, r)
	if !ok {
		return
	}
	comment, _ := decodeComment(r)
	row, err := h.Service.Close(r.Context(), id, actor, httputil.ClientIP(r), comment)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, incident.View(row))
}

// StartCorrectiveAction godoc
// @Summary      Move incident to corrective action (P2 stub)
// @Tags         workflow
// @Produce      json
// @Param        id   path      string  true  "id"
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/start-corrective-action [post]
func (h *Handler) StartCorrectiveAction(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := h.actorID(w, r)
	if !ok {
		return
	}
	comment, _ := decodeComment(r)
	row, err := h.Service.StartCorrectiveAction(r.Context(), id, actor, httputil.ClientIP(r), comment)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, incident.View(row))
}

// Timeline godoc
// @Summary      Incident workflow timeline
// @Tags         workflow
// @Produce      json
// @Param        id   path      string  true  "id"
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/timeline [get]
func (h *Handler) Timeline(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := h.actorID(w, r)
	if !ok {
		return
	}
	items, err := h.Service.Timeline(id, actor)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{
			"id":         it.ID,
			"fromStatus": it.FromStatus,
			"toStatus":   it.ToStatus,
			"actorId":    it.ActorID,
			"comment":    it.Comment,
			"createdAt":  it.CreatedAt,
		})
	}
	_ = response.Success(w, http.StatusOK, out)
}
