package middleware

import (
	"net/http"
	"strings"

	"aegis/pkg/authctx"
	"aegis/pkg/response"
)

type AccessParser interface {
	ParseAccess(token string) (authctx.Principal, error)
}

func RequireAuth(parser AccessParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r)
			if raw == "" {
				_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing access token")
				return
			}
			p, err := parser.ParseAccess(raw)
			if err != nil {
				_ = response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired access token")
				return
			}
			ctx := authctx.WithPrincipal(r.Context(), p)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if strings.HasPrefix(h, prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
