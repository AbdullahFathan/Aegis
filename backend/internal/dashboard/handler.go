package dashboard

import (
	"net/http"
	"time"

	"aegis/internal/incident"
	"aegis/pkg/response"

	"github.com/google/uuid"
)

type Handler struct {
	Service *Service
}

func parseFilter(r *http.Request) (Filter, error) {
	var f Filter
	q := r.URL.Query()
	f.Category = q.Get("category")
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

// Summary godoc
// @Summary      Dashboard KPI summary
// @Tags         dashboard
// @Produce      json
// @Param        from        query     string  false  "RFC3339"
// @Param        to          query     string  false  "RFC3339"
// @Param        locationId  query     string  false  "location id"
// @Param        category    query     string  false  "category"
// @Success      200         {object}  response.Envelope
// @Failure      401         {object}  response.Envelope
// @Failure      403         {object}  response.Envelope
// @Security     BearerAuth
// @Router       /dashboard/summary [get]
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	out, err := h.Service.Summary(f)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, out)
}

// Trends godoc
// @Summary      Incident trends 12 months
// @Tags         dashboard
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /dashboard/trends [get]
func (h *Handler) Trends(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	out, err := h.Service.Trends(f)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, out)
}

// Heatmap godoc
// @Summary      Heatmap site x category
// @Tags         dashboard
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /dashboard/heatmap [get]
func (h *Handler) Heatmap(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	out, err := h.Service.Heatmap(f)
	if err != nil {
		incident.WriteErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, out)
}
