package rca

import (
	"encoding/json"
	"net/http"

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

type upsertBody struct {
	Timeline          string             `json:"timeline"`
	HumanFactor       string             `json:"humanFactor"`
	EnvironmentFactor string             `json:"environmentFactor"`
	EquipmentFactor   string             `json:"equipmentFactor"`
	FiveWhys          []database.FiveWhy `json:"fiveWhys"`
	Fishbone          database.Fishbone  `json:"fishbone"`
	Completed         bool               `json:"completed"`
}

type templateBody struct {
	Timeline          string             `json:"timeline"`
	HumanFactor       string             `json:"humanFactor"`
	EnvironmentFactor string             `json:"environmentFactor"`
	EquipmentFactor   string             `json:"equipmentFactor"`
	FiveWhys          []database.FiveWhy `json:"fiveWhys"`
	Fishbone          database.Fishbone  `json:"fishbone"`
}

// Get godoc
// @Summary      Get RCA for incident
// @Tags         rca
// @Produce      json
// @Param        id   path      string  true  "incident id"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/rca [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := actorIncident(w, r)
	if !ok {
		return
	}
	row, err := h.Service.Get(id, actor)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, viewRCA(row))
}

// Upsert godoc
// @Summary      Create or update RCA
// @Tags         rca
// @Accept       json
// @Produce      json
// @Param        id       path      string      true  "incident id"
// @Param        request  body      upsertBody  true  "rca"
// @Success      200      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/rca [put]
func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	actor, id, ok := actorIncident(w, r)
	if !ok {
		return
	}
	var body upsertBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	row, err := h.Service.Upsert(r.Context(), id, UpsertInput{
		Timeline: body.Timeline, HumanFactor: body.HumanFactor,
		EnvironmentFactor: body.EnvironmentFactor, EquipmentFactor: body.EquipmentFactor,
		FiveWhys: body.FiveWhys, Fishbone: body.Fishbone, Completed: body.Completed,
	}, actor, httputil.ClientIP(r))
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, viewRCA(row))
}

// GetTemplate godoc
// @Summary      Get RCA template by category
// @Tags         rca
// @Produce      json
// @Param        category  path      string  true  "incident category"
// @Success      200       {object}  response.Envelope
// @Security     BearerAuth
// @Router       /rca-templates/{category} [get]
func (h *Handler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	cat := database.IncidentCategory(chi.URLParam(r, "category"))
	row, err := h.Service.GetTemplate(cat)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, viewTemplate(row))
}

// PutTemplate godoc
// @Summary      Save RCA template by category
// @Tags         rca
// @Accept       json
// @Produce      json
// @Param        category  path      string        true  "incident category"
// @Param        request   body      templateBody  true  "template"
// @Success      200       {object}  response.Envelope
// @Failure      403       {object}  response.Envelope
// @Security     BearerAuth
// @Router       /rca-templates/{category} [put]
func (h *Handler) PutTemplate(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	cat := database.IncidentCategory(chi.URLParam(r, "category"))
	var body templateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	row, err := h.Service.PutTemplate(r.Context(), cat, database.RCATemplatePayload{
		Timeline: body.Timeline, HumanFactor: body.HumanFactor,
		EnvironmentFactor: body.EnvironmentFactor, EquipmentFactor: body.EquipmentFactor,
		FiveWhys: body.FiveWhys, Fishbone: body.Fishbone,
	}, actor, httputil.ClientIP(r))
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, viewTemplate(row))
}

func actorIncident(w http.ResponseWriter, r *http.Request) (authctx.Principal, uuid.UUID, bool) {
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

func viewRCA(row database.RootCauseAnalysis) map[string]any {
	return map[string]any{
		"id":                row.ID,
		"incidentId":        row.IncidentID,
		"timeline":          row.Timeline,
		"humanFactor":       row.HumanFactor,
		"environmentFactor": row.EnvironmentFactor,
		"equipmentFactor":   row.EquipmentFactor,
		"fiveWhys":          row.FiveWhys,
		"fishbone":          row.Fishbone,
		"investigatorId":    row.InvestigatorID,
		"completedAt":       row.CompletedAt,
		"updatedAt":         row.UpdatedAt,
	}
}

func viewTemplate(row database.RCATemplate) map[string]any {
	return map[string]any{
		"id":       row.ID,
		"category": row.Category,
		"payload":  row.Payload,
	}
}
