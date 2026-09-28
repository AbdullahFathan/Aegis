package middleware

import (
	"net/http"

	"aegis/pkg/authctx"
	"aegis/pkg/rbac"
	"aegis/pkg/response"
)

func RequirePermission(perm rbac.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := authctx.PrincipalFrom(r.Context())
			if !ok {
				_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing access token")
				return
			}
			if !rbac.Has(p.Role, perm) {
				_ = response.Error(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
