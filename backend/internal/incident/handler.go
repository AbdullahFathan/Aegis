package incident

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	Category          string             `json:"category"`
	Severity          string             `json:"severity"`
	IncidentDatetime  string             `json:"incidentDatetime"`
	LocationID        string             `json:"locationId"`
	AreaID            *string            `json:"areaId"`
	HasVictim         bool               `json:"hasVictim"`
	VictimName        *string            `json:"victimName"`
	VictimPosition    *string            `json:"victimPosition"`
	InjuryDescription *string            `json:"injuryDescription"`
	InitialTreatment  *string            `json:"initialTreatment"`
	Witnesses         []database.Witness `json:"witnesses"`
}

type patchBody struct {
	Status            *string             `json:"status"`
	Reason            *string             `json:"reason"`
	Title             *string             `json:"title"`
	Description       *string             `json:"description"`
	Category          *string             `json:"category"`
	Severity          *string             `json:"severity"`
	IncidentDatetime  *string             `json:"incidentDatetime"`
	LocationID        *string             `json:"locationId"`
	AreaID            *string             `json:"areaId"`
	HasVictim         *bool               `json:"hasVictim"`
	VictimName        *string             `json:"victimName"`
	VictimPosition    *string             `json:"victimPosition"`
	InjuryDescription *string             `json:"injuryDescription"`
	InitialTreatment  *string             `json:"initialTreatment"`
	Witnesses         *[]database.Witness `json:"witnesses"`
}

// List godoc
// @Summary      List incidents
// @Tags         incidents
// @Produce      json
// @Param        status      query     string  false  "status"
// @Param        category    query     string  false  "category"
// @Param        locationId  query     string  false  "location id"
// @Param        from        query     string  false  "from RFC3339"
// @Param        to          query     string  false  "to RFC3339"
// @Param        page        query     int     false  "page"
// @Param        pageSize    query     int     false  "page size"
// @Success      200         {object}  response.Envelope
// @Failure      401         {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	q := r.URL.Query()
	f := ListFilter{
		Status:   q.Get("status"),
		Category: q.Get("category"),
	}
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.PageSize, _ = strconv.Atoi(q.Get("pageSize"))
	if v := q.Get("locationId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid locationId")
			return
		}
		f.LocationID = &id
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
	items, total, err := h.Service.List(actor, f)
	if err != nil {
		WriteErr(w, err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, View(it))
	}
	page, pageSize := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	_ = response.Success(w, http.StatusOK, map[string]any{
		"items": out, "page": page, "pageSize": pageSize, "total": total,
	})
}

// Create godoc
// @Summary      Create incident draft
// @Tags         incidents
// @Accept       json
// @Produce      json
// @Param        request  body      createBody  true  "incident"
// @Success      201      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	var body createBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	when, err := time.Parse(time.RFC3339, body.IncidentDatetime)
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid incidentDatetime")
		return
	}
	locID, err := uuid.Parse(body.LocationID)
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid locationId")
		return
	}
	in := CreateInput{
		Title: body.Title, Description: body.Description,
		Category:         database.IncidentCategory(body.Category),
		Severity:         database.Severity(body.Severity),
		IncidentDatetime: when, LocationID: locID, HasVictim: body.HasVictim,
		VictimName: body.VictimName, VictimPosition: body.VictimPosition,
		InjuryDescription: body.InjuryDescription, InitialTreatment: body.InitialTreatment,
		Witnesses: body.Witnesses,
	}
	if body.AreaID != nil && *body.AreaID != "" {
		aid, err := uuid.Parse(*body.AreaID)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid areaId")
			return
		}
		in.AreaID = &aid
	}
	row, err := h.Service.Create(r.Context(), in, actor, httputil.ClientIP(r))
	if err != nil {
		WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusCreated, View(row))
}

// Get godoc
// @Summary      Get incident
// @Tags         incidents
// @Produce      json
// @Param        id   path      string  true  "id"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
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
	row, err := h.Service.Get(id, actor)
	if err != nil {
		WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, View(row))
}

// Patch godoc
// @Summary      Update incident
// @Tags         incidents
// @Accept       json
// @Produce      json
// @Param        id       path      string     true  "id"
// @Param        request  body      patchBody  true  "fields"
// @Success      200      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id} [patch]
func (h *Handler) Patch(w http.ResponseWriter, r *http.Request) {
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
	var body patchBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	in := PatchInput{
		Status: body.Status, Reason: body.Reason, Title: body.Title, Description: body.Description,
		HasVictim: body.HasVictim, VictimName: body.VictimName, VictimPosition: body.VictimPosition,
		InjuryDescription: body.InjuryDescription, InitialTreatment: body.InitialTreatment, Witnesses: body.Witnesses,
	}
	if body.Category != nil {
		c := database.IncidentCategory(*body.Category)
		in.Category = &c
	}
	if body.Severity != nil {
		c := database.Severity(*body.Severity)
		in.Severity = &c
	}
	if body.IncidentDatetime != nil {
		t, err := time.Parse(time.RFC3339, *body.IncidentDatetime)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid incidentDatetime")
			return
		}
		in.IncidentDatetime = &t
	}
	if body.LocationID != nil {
		lid, err := uuid.Parse(*body.LocationID)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid locationId")
			return
		}
		in.LocationID = &lid
	}
	if body.AreaID != nil {
		if *body.AreaID == "" {
			in.AreaID = nil
		} else {
			aid, err := uuid.Parse(*body.AreaID)
			if err != nil {
				_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid areaId")
				return
			}
			in.AreaID = &aid
		}
	}
	row, err := h.Service.Patch(r.Context(), id, in, actor, httputil.ClientIP(r))
	if err != nil {
		WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, View(row))
}

// Submit godoc
// @Summary      Submit incident draft
// @Tags         incidents
// @Produce      json
// @Param        id   path      string  true  "id"
// @Success      200  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      409  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /incidents/{id}/submit [post]
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
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
	row, err := h.Service.Submit(r.Context(), id, actor, httputil.ClientIP(r))
	if err != nil {
		WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, View(row))
}

func WriteErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", errDetail(err, ErrValidation, "invalid payload"))
	case errors.Is(err, ErrForbidden):
		_ = response.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
	case errors.Is(err, ErrNotFound):
		_ = response.Error(w, http.StatusNotFound, "NOT_FOUND", "not found")
	case errors.Is(err, ErrConflict):
		_ = response.Error(w, http.StatusConflict, "CONFLICT", "conflict")
	case errors.Is(err, ErrIllegal):
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", errDetail(err, ErrIllegal, "illegal transition"))
	default:
		_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
}

func errDetail(err, sentinel error, fallback string) string {
	prefix := sentinel.Error() + ": "
	if strings.HasPrefix(err.Error(), prefix) {
		return strings.TrimPrefix(err.Error(), prefix)
	}
	if err.Error() == sentinel.Error() {
		return fallback
	}
	return err.Error()
}
