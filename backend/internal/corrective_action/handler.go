package correctiveaction

import (
	"encoding/json"
	"net/http"
	"time"

	"aegis/internal/incident"
	"aegis/pkg/authctx"
	"aegis/pkg/database"
	"aegis/pkg/httputil"
	"aegis/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	Service *Service
}

type createBody struct {
	Description string `json:"description"`
	ActionType  string `json:"actionType"`
	Priority    string `json:"priority"`
	AssigneeID  string `json:"assigneeId"`
	DueDate     string `json:"dueDate"`
}

type patchBody struct {
	Description     *string `json:"description"`
	ActionType      *string `json:"actionType"`
	Priority        *string `json:"priority"`
	AssigneeID      *string `json:"assigneeId"`
	DueDate         *string `json:"dueDate"`
	Status          *string `json:"status"`
	CompletionNotes *string `json:"completionNotes"`
}

// ListByIncident godoc
// @Summary      List corrective actions for an incident
// @Tags         corrective-actions
// @Produce      json
// @Param        id   path      string  true  "incident id"
// @Success      200  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/corrective-actions [get]
func (h *Handler) ListByIncident(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := actorUUID(w, r, "id")
	if !ok {
		return
	}
	items, err := h.Service.ListByIncident(id, actor)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, viewList(items))
}

// Create godoc
// @Summary      Create corrective action
// @Tags         corrective-actions
// @Accept       json
// @Produce      json
// @Param        id       path      string      true  "incident id"
// @Param        request  body      createBody  true  "ca"
// @Success      201      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/corrective-actions [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := actorUUID(w, r, "id")
	if !ok {
		return
	}
	var body createBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	assignee, err := uuid.Parse(body.AssigneeID)
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid assigneeId")
		return
	}
	due, err := parseDate(body.DueDate)
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid dueDate")
		return
	}
	row, err := h.Service.Create(r.Context(), id, CreateInput{
		Description: body.Description,
		ActionType:  database.ActionType(body.ActionType),
		Priority:    database.CAPriority(body.Priority),
		AssigneeID:  assignee,
		DueDate:     due,
	}, actor, httputil.ClientIP(r))
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusCreated, viewCA(row))
}

// Patch godoc
// @Summary      Update corrective action
// @Tags         corrective-actions
// @Accept       json
// @Produce      json
// @Param        id       path      string     true  "ca id"
// @Param        request  body      patchBody  true  "patch"
// @Success      200      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /corrective-actions/{id} [patch]
func (h *Handler) Patch(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := actorUUID(w, r, "id")
	if !ok {
		return
	}
	var body patchBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	in := PatchInput{Description: body.Description, CompletionNotes: body.CompletionNotes}
	if body.ActionType != nil {
		v := database.ActionType(*body.ActionType)
		in.ActionType = &v
	}
	if body.Priority != nil {
		v := database.CAPriority(*body.Priority)
		in.Priority = &v
	}
	if body.Status != nil {
		v := database.CAStatus(*body.Status)
		in.Status = &v
	}
	if body.AssigneeID != nil {
		uid, err := uuid.Parse(*body.AssigneeID)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid assigneeId")
			return
		}
		in.AssigneeID = &uid
	}
	if body.DueDate != nil {
		due, err := parseDate(*body.DueDate)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid dueDate")
			return
		}
		in.DueDate = &due
	}
	row, err := h.Service.Patch(r.Context(), id, in, actor, httputil.ClientIP(r))
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, viewCA(row))
}

// Verify godoc
// @Summary      Verify completed corrective action
// @Tags         corrective-actions
// @Produce      json
// @Param        id   path      string  true  "ca id"
// @Success      200  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /corrective-actions/{id}/verify [post]
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := actorUUID(w, r, "id")
	if !ok {
		return
	}
	row, err := h.Service.Verify(r.Context(), id, actor, httputil.ClientIP(r))
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, viewCA(row))
}

// Tracker godoc
// @Summary      List corrective actions across incidents
// @Tags         corrective-actions
// @Produce      json
// @Param        status      query     string  false  "status"
// @Param        assigneeId  query     string  false  "assignee"
// @Param        priority    query     string  false  "priority"
// @Param        locationId  query     string  false  "location"
// @Success      200         {object}  response.Envelope
// @Security     BearerAuth
// @Router       /corrective-actions [get]
func (h *Handler) Tracker(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	q := r.URL.Query()
	f := ListFilter{Status: q.Get("status"), Priority: q.Get("priority")}
	if v := q.Get("assigneeId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid assigneeId")
			return
		}
		f.AssigneeID = &id
	}
	if v := q.Get("locationId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid locationId")
			return
		}
		f.LocationID = &id
	}
	items, err := h.Service.Tracker(actor, f)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, viewList(items))
}

func actorUUID(w http.ResponseWriter, r *http.Request, param string) (authctx.Principal, uuid.UUID, bool) {
	actor, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return authctx.Principal{}, uuid.Nil, false
	}
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid id")
		return authctx.Principal{}, uuid.Nil, false
	}
	return actor, id, true
}

func parseDate(raw string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", raw)
}

func viewCA(row database.CorrectiveAction) map[string]any {
	return map[string]any{
		"id":              row.ID,
		"incidentId":      row.IncidentID,
		"description":     row.Description,
		"actionType":      row.ActionType,
		"priority":        row.Priority,
		"status":          row.Status,
		"assigneeId":      row.AssigneeID,
		"dueDate":         row.DueDate,
		"completionNotes": row.CompletionNotes,
		"completedAt":     row.CompletedAt,
		"verifiedById":    row.VerifiedByID,
		"verifiedAt":      row.VerifiedAt,
	}
}

func viewList(items []database.CorrectiveAction) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, viewCA(it))
	}
	return out
}
