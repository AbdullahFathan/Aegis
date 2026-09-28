package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aegis/pkg/middleware"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRecovererDoesNotLeakStack(t *testing.T) {
	log := zap.NewNop()
	h := middleware.Recoverer(log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("secret-stack-marker /pkg/middleware/middleware.go:99")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	s := string(body)
	require.Contains(t, s, `"success":false`)
	require.Contains(t, s, `"code":"INTERNAL"`)
	require.NotContains(t, s, "secret-stack-marker")
	require.NotContains(t, strings.ToLower(s), "goroutine")
	require.NotContains(t, s, "middleware.go")
	require.NotContains(t, s, "panic:")
}
