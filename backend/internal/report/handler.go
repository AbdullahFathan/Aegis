package report

import (
	"encoding/json"
	"net/http"
	"time"

	"aegis/internal/corrective_action"
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

func parseQuery(r *http.Request) (QueryFilter, error) {
	var f QueryFilter
	q := r.URL.Query()
	f.Category = q.Get("category")
	f.Confidential = q.Get("confidential") == "true"
	if v := q.Get("locationId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return f, incident.WrapValidation("invalid locationId")
		}
		f.LocationID = &id
	}
	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, incident.WrapValidation("invalid from")
		}
		f.From = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return f, incident.WrapValidation("invalid to")
		}
		f.To = &t
	}
	return f, nil
}

func writeCSV(w http.ResponseWriter, filename string, rows [][]string) {
	b, err := EncodeCSV(rows)
	if err != nil {
		_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "csv encode failed")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func actorOr401(w http.ResponseWriter, r *http.Request) (authctx.Principal, bool) {
	actor, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return authctx.Principal{}, false
	}
	return actor, true
}

func formatOf(r *http.Request) string {
	f := r.URL.Query().Get("format")
	if f == "" {
		return "json"
	}
	return f
}

func (h *Handler) maybePDF(w http.ResponseWriter, r *http.Request, typ database.ReportType, f QueryFilter, incidentID *uuid.UUID) bool {
	if formatOf(r) != "pdf" {
		return false
	}
	actor, ok := actorOr401(w, r)
	if !ok {
		return true
	}
	job, err := h.Service.EnqueuePDF(r.Context(), actor, typ, f, incidentID)
	if err != nil {
		incident.WriteErr(w, err)
		return true
	}
	_ = response.Success(w, http.StatusAccepted, map[string]any{
		"jobId":  job.ID.String(),
		"status": job.Status,
	})
	return true
}

// Monthly godoc
// @Summary      Monthly incident recap
// @Tags         reports
// @Produce      json
// @Param        format  query     string  false  "json|csv|pdf"
// @Success      200     {object}  response.Envelope
// @Success      202     {object}  response.Envelope
// @Failure      401     {object}  response.Envelope
// @Failure      403     {object}  response.Envelope
// @Failure      422     {object}  response.Envelope
// @Security     BearerAuth
// @Router       /reports/monthly [get]
func (h *Handler) Monthly(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOr401(w, r)
	if !ok {
		return
	}
	f, err := parseQuery(r)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	if h.maybePDF(w, r, database.ReportMonthly, f, nil) {
		return
	}
	items, err := h.Service.listIncidents(r.Context(), actor, f)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	if formatOf(r) == "csv" {
		writeCSV(w, "monthly.csv", MonthlyCSV(items))
		return
	}
	views := make([]map[string]any, 0, len(items))
	for _, it := range items {
		views = append(views, incident.View(it))
	}
	_ = response.Success(w, http.StatusOK, views)
}

// LTIFR godoc
// @Summary      LTIFR and TRIFR
// @Tags         reports
// @Produce      json
// @Param        format  query     string  false  "json|csv|pdf"
// @Success      200     {object}  response.Envelope
// @Failure      401     {object}  response.Envelope
// @Failure      403     {object}  response.Envelope
// @Failure      422     {object}  response.Envelope
// @Security     BearerAuth
// @Router       /reports/ltifr [get]
func (h *Handler) LTIFR(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOr401(w, r)
	if !ok {
		return
	}
	f, err := parseQuery(r)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	if h.maybePDF(w, r, database.ReportLTIFR, f, nil) {
		return
	}
	out, err := h.Service.LTIFR(r.Context(), actor, f)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	if formatOf(r) == "csv" {
		writeCSV(w, "ltifr.csv", LTIFRCSV(out))
		return
	}
	_ = response.Success(w, http.StatusOK, out)
}

// CorrectiveActions godoc
// @Summary      Corrective action status report
// @Tags         reports
// @Produce      json
// @Param        format  query     string  false  "json|csv|pdf"
// @Success      200     {object}  response.Envelope
// @Failure      401     {object}  response.Envelope
// @Failure      403     {object}  response.Envelope
// @Failure      422     {object}  response.Envelope
// @Security     BearerAuth
// @Router       /reports/corrective-actions [get]
func (h *Handler) CorrectiveActions(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOr401(w, r)
	if !ok {
		return
	}
	f, err := parseQuery(r)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	if h.maybePDF(w, r, database.ReportCAStatus, f, nil) {
		return
	}
	items, err := h.Service.listCAs(r.Context(), actor, f)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	if formatOf(r) == "csv" {
		writeCSV(w, "corrective-actions.csv", CACSV(items))
		return
	}
	_ = response.Success(w, http.StatusOK, correctiveaction.ViewList(items))
}

// Investigation godoc
// @Summary      Investigation report PDF
// @Tags         reports
// @Param        id      path      string  true  "incident id"
// @Param        format  query     string  false  "pdf"
// @Success      202     {object}  response.Envelope
// @Failure      401     {object}  response.Envelope
// @Failure      403     {object}  response.Envelope
// @Failure      422     {object}  response.Envelope
// @Security     BearerAuth
// @Router       /reports/investigation/{id} [get]
func (h *Handler) Investigation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid id")
		return
	}
	f, err := parseQuery(r)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	actor, ok := actorOr401(w, r)
	if !ok {
		return
	}
	job, err := h.Service.EnqueuePDF(r.Context(), actor, database.ReportInvestigation, f, &id)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusAccepted, map[string]any{
		"jobId":  job.ID.String(),
		"status": job.Status,
	})
}

// Job godoc
// @Summary      Poll generated report job
// @Tags         reports
// @Produce      json
// @Param        id   path      string  true  "job id"
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /reports/jobs/{id} [get]
func (h *Handler) Job(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOr401(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid id")
		return
	}
	row, url, err := h.Service.GetJob(r.Context(), id, actor)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, map[string]any{
		"id":          row.ID.String(),
		"type":        row.Type,
		"status":      row.Status,
		"storedKey":   row.StoredKey,
		"downloadUrl": url,
		"error":       row.ErrorMessage,
	})
}

// Archive godoc
// @Summary      List archived generated reports
// @Tags         reports
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /reports [get]
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOr401(w, r)
	if !ok {
		return
	}
	rows, err := h.Service.ListArchive(r.Context(), actor)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, rows)
}

type workHoursBody struct {
	LocationID  *string `json:"locationId"`
	PeriodStart string  `json:"periodStart"`
	PeriodEnd   string  `json:"periodEnd"`
	Hours       float64 `json:"hours"`
}

// PutWorkHours godoc
// @Summary      Upsert work hours for a period
// @Tags         reports
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /work-hours [put]
func (h *Handler) PutWorkHours(w http.ResponseWriter, r *http.Request) {
	var body workHoursBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	start, err := time.Parse(time.RFC3339, body.PeriodStart)
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid periodStart")
		return
	}
	end, err := time.Parse(time.RFC3339, body.PeriodEnd)
	if err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid periodEnd")
		return
	}
	var locID *uuid.UUID
	if body.LocationID != nil && *body.LocationID != "" {
		id, err := uuid.Parse(*body.LocationID)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid locationId")
			return
		}
		locID = &id
	}
	row, err := h.Service.UpsertWorkHours(r.Context(), locID, start, end, body.Hours)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, row)
}

// ListWorkHours godoc
// @Summary      List work hours rows
// @Tags         reports
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      422  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /work-hours [get]
func (h *Handler) ListWorkHours(w http.ResponseWriter, r *http.Request) {
	var locID *uuid.UUID
	if v := r.URL.Query().Get("locationId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid locationId")
			return
		}
		locID = &id
	}
	rows, err := h.Service.ListWorkHours(r.Context(), locID)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, rows)
}
