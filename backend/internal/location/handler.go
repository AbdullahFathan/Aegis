package location

import (
	"encoding/json"
	"errors"
	"net/http"

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

type regionBody struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type locationBody struct {
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	Type         string  `json:"type"`
	RegionID     *string `json:"regionId"`
	SupervisorID string  `json:"supervisorId"`
	HSEOfficerID string  `json:"hseOfficerId"`
}

type areaBody struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func regionView(r database.Region) map[string]any {
	return map[string]any{"id": r.ID, "name": r.Name, "code": r.Code}
}

func locationView(l database.Location) map[string]any {
	areas := make([]map[string]any, 0, len(l.Areas))
	for _, a := range l.Areas {
		areas = append(areas, areaView(a))
	}
	return map[string]any{
		"id":           l.ID,
		"name":         l.Name,
		"code":         l.Code,
		"type":         l.Type,
		"regionId":     l.RegionID,
		"supervisorId": l.SupervisorID,
		"hseOfficerId": l.HSEOfficerID,
		"isActive":     l.IsActive,
		"areas":        areas,
	}
}

func areaView(a database.Area) map[string]any {
	return map[string]any{"id": a.ID, "name": a.Name, "code": a.Code, "locationId": a.LocationID}
}

// ListRegions godoc
// @Summary      List regions
// @Tags         locations
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /regions [get]
func (h *Handler) ListRegions(w http.ResponseWriter, r *http.Request) {
	items, err := h.Service.ListRegions()
	if err != nil {
		_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, regionView(it))
	}
	_ = response.Success(w, http.StatusOK, out)
}

// CreateRegion godoc
// @Summary      Create region
// @Tags         locations
// @Accept       json
// @Produce      json
// @Param        request  body      regionBody  true  "region"
// @Success      201      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /regions [post]
func (h *Handler) CreateRegion(w http.ResponseWriter, r *http.Request) {
	var body regionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	actor, _ := authctx.PrincipalFrom(r.Context())
	row, err := h.Service.CreateRegion(r.Context(), body.Name, body.Code, actor, httputil.ClientIP(r))
	if err != nil {
		writeLocErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusCreated, regionView(row))
}

// ListLocations godoc
// @Summary      List locations
// @Tags         locations
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /locations [get]
func (h *Handler) ListLocations(w http.ResponseWriter, r *http.Request) {
	items, err := h.Service.ListLocations()
	if err != nil {
		_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, locationView(it))
	}
	_ = response.Success(w, http.StatusOK, out)
}

// CreateLocation godoc
// @Summary      Create location
// @Tags         locations
// @Accept       json
// @Produce      json
// @Param        request  body      locationBody  true  "location"
// @Success      201      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /locations [post]
func (h *Handler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	var body locationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	in, err := parseLocationBody(body, true)
	if err != nil {
		writeLocErr(w, err)
		return
	}
	actor, _ := authctx.PrincipalFrom(r.Context())
	row, err := h.Service.CreateLocation(r.Context(), in, actor, httputil.ClientIP(r))
	if err != nil {
		writeLocErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusCreated, locationView(row))
}

// PatchLocation godoc
// @Summary      Update location
// @Tags         locations
// @Accept       json
// @Produce      json
// @Param        id       path      string        true  "location id"
// @Param        request  body      locationBody  true  "fields"
// @Success      200      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /locations/{id} [patch]
func (h *Handler) PatchLocation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid id")
		return
	}
	var body locationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	in, err := parseLocationBody(body, false)
	if err != nil {
		writeLocErr(w, err)
		return
	}
	actor, _ := authctx.PrincipalFrom(r.Context())
	row, err := h.Service.PatchLocation(r.Context(), id, in, actor, httputil.ClientIP(r))
	if err != nil {
		writeLocErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, locationView(row))
}

// DeleteLocation godoc
// @Summary      Deactivate location
// @Tags         locations
// @Produce      json
// @Param        id   path      string  true  "location id"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /locations/{id} [delete]
func (h *Handler) DeleteLocation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid id")
		return
	}
	actor, _ := authctx.PrincipalFrom(r.Context())
	row, err := h.Service.Deactivate(r.Context(), id, actor, httputil.ClientIP(r))
	if err != nil {
		writeLocErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, locationView(row))
}

// ListAreas godoc
// @Summary      List areas
// @Tags         locations
// @Produce      json
// @Param        id   path      string  true  "location id"
// @Success      200  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /locations/{id}/areas [get]
func (h *Handler) ListAreas(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid id")
		return
	}
	items, err := h.Service.ListAreas(id)
	if err != nil {
		writeLocErr(w, err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, areaView(it))
	}
	_ = response.Success(w, http.StatusOK, out)
}

// CreateArea godoc
// @Summary      Create area
// @Tags         locations
// @Accept       json
// @Produce      json
// @Param        id       path      string    true  "location id"
// @Param        request  body      areaBody  true  "area"
// @Success      201      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /locations/{id}/areas [post]
func (h *Handler) CreateArea(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid id")
		return
	}
	var body areaBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	actor, _ := authctx.PrincipalFrom(r.Context())
	row, err := h.Service.CreateArea(r.Context(), id, body.Name, body.Code, actor, httputil.ClientIP(r))
	if err != nil {
		writeLocErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusCreated, areaView(row))
}

func parseLocationBody(body locationBody, create bool) (LocationInput, error) {
	in := LocationInput{
		Name: body.Name,
		Code: body.Code,
		Type: database.LocationType(body.Type),
	}
	if body.SupervisorID != "" {
		id, err := uuid.Parse(body.SupervisorID)
		if err != nil {
			return LocationInput{}, ErrValidation
		}
		in.SupervisorID = id
	} else if create {
		return LocationInput{}, ErrValidation
	}
	if body.HSEOfficerID != "" {
		id, err := uuid.Parse(body.HSEOfficerID)
		if err != nil {
			return LocationInput{}, ErrValidation
		}
		in.HSEOfficerID = id
	} else if create {
		return LocationInput{}, ErrValidation
	}
	if body.RegionID != nil && *body.RegionID != "" {
		id, err := uuid.Parse(*body.RegionID)
		if err != nil {
			return LocationInput{}, ErrValidation
		}
		in.RegionID = &id
	}
	return in, nil
}

func writeLocErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid location payload")
	case errors.Is(err, ErrConflict):
		_ = response.Error(w, http.StatusConflict, "CONFLICT", "code already exists")
	case errors.Is(err, ErrNotFound):
		_ = response.Error(w, http.StatusNotFound, "NOT_FOUND", "location not found")
	default:
		_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
}
