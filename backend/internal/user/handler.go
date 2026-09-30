package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"aegis/internal/auth"
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
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type patchBody struct {
	Name   *string `json:"name"`
	Role   *string `json:"role"`
	Status *string `json:"status"`
}

func view(u database.User) auth.UserView {
	return auth.UserView{
		ID:     u.ID.String(),
		Email:  u.Email,
		Name:   u.Name,
		Role:   string(u.Role),
		Status: string(u.Status),
	}
}

// List godoc
// @Summary      List users
// @Tags         users
// @Produce      json
// @Param        page      query     int     false  "page"
// @Param        pageSize  query     int     false  "page size"
// @Param        role      query     string  false  "role"
// @Param        status    query     string  false  "status"
// @Success      200       {object}  response.Envelope
// @Failure      401       {object}  response.Envelope
// @Failure      403       {object}  response.Envelope
// @Security     BearerAuth
// @Router       /users [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	items, total, err := h.Service.List(r.Context(), page, pageSize, r.URL.Query().Get("role"), r.URL.Query().Get("status"))
	if err != nil {
		if response.WriteTimeout(w, err) {
			return
		}
		_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	out := make([]auth.UserView, 0, len(items))
	for _, u := range items {
		out = append(out, view(u))
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	_ = response.Success(w, http.StatusOK, map[string]any{
		"items":    out,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

// Create godoc
// @Summary      Create user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      createBody  true  "user"
// @Success      201      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      409      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /users [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body createBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	actor, _ := authctx.PrincipalFrom(r.Context())
	u, err := h.Service.Create(r.Context(), CreateInput{
		Email:    body.Email,
		Password: body.Password,
		Name:     body.Name,
		Role:     database.Role(body.Role),
		Status:   database.UserStatus(body.Status),
	}, actor, httputil.ClientIP(r))
	if err != nil {
		writeUserErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusCreated, view(u))
}

// Patch godoc
// @Summary      Update user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id       path      string     true  "user id"
// @Param        request  body      patchBody  true  "fields"
// @Success      200      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      403      {object}  response.Envelope
// @Failure      404      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Security     BearerAuth
// @Router       /users/{id} [patch]
func (h *Handler) Patch(w http.ResponseWriter, r *http.Request) {
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
	in := PatchInput{Name: body.Name}
	if body.Role != nil {
		role := database.Role(*body.Role)
		in.Role = &role
	}
	if body.Status != nil {
		st := database.UserStatus(*body.Status)
		in.Status = &st
	}
	actor, _ := authctx.PrincipalFrom(r.Context())
	u, err := h.Service.Patch(r.Context(), id, in, actor, httputil.ClientIP(r))
	if err != nil {
		writeUserErr(w, err)
		return
	}
	_ = response.Success(w, http.StatusOK, view(u))
}

func writeUserErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid user payload")
	case errors.Is(err, ErrEmailTaken):
		_ = response.Error(w, http.StatusConflict, "CONFLICT", "email already exists")
	case errors.Is(err, ErrNotFound):
		_ = response.Error(w, http.StatusNotFound, "NOT_FOUND", "user not found")
	default:
		if response.WriteTimeout(w, err) {
			return
		}
		_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
}
