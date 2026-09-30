package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"aegis/pkg/authctx"
	"aegis/pkg/response"
)

type Handler struct {
	Service      *Service
	CookieSecure bool
	CookieMaxAge time.Duration
}

// Login godoc
// @Summary      Login
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "credentials"
// @Success      200      {object}  response.Envelope
// @Failure      401      {object}  response.Envelope
// @Failure      422      {object}  response.Envelope
// @Failure      429      {object}  response.Envelope
// @Router       /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		_ = response.Error(w, http.StatusUnprocessableEntity, "VALIDATION", "email and password are required")
		return
	}
	out, refresh, err := h.Service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", ErrInvalidCredentials.Error())
			return
		}
		if response.WriteTimeout(w, err) {
			return
		}
		_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	h.setRefreshCookie(w, refresh)
	_ = response.Success(w, http.StatusOK, out)
}

// Refresh godoc
// @Summary      Refresh access token
// @Tags         auth
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(cookieName)
	raw := ""
	if err == nil {
		raw = c.Value
	}
	out, refresh, err := h.Service.Refresh(r.Context(), raw)
	if err != nil {
		h.clearRefreshCookie(w)
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
		return
	}
	h.setRefreshCookie(w, refresh)
	_ = response.Success(w, http.StatusOK, out)
}

// Logout godoc
// @Summary      Logout
// @Tags         auth
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Router       /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	raw := ""
	if c, err := r.Cookie(cookieName); err == nil {
		raw = c.Value
	}
	_ = h.Service.Logout(r.Context(), raw)
	h.clearRefreshCookie(w)
	_ = response.Success(w, http.StatusOK, map[string]bool{"loggedOut": true})
}

// Me godoc
// @Summary      Current user
// @Tags         auth
// @Produce      json
// @Success      200  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Security     BearerAuth
// @Router       /auth/me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	p, ok := authctx.PrincipalFrom(r.Context())
	if !ok {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing access token")
		return
	}
	view, err := h.Service.Me(r.Context(), p.ID)
	if err != nil {
		_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired access token")
		return
	}
	_ = response.Success(w, http.StatusOK, view)
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(h.CookieMaxAge.Seconds()),
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
