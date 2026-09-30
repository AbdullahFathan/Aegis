package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aegis/pkg/middleware"
	"aegis/pkg/response"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestTimeoutWritesGatewayTimeout(t *testing.T) {
	h := middleware.Timeout(20 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "late")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusGatewayTimeout, rec.Code)
	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	s := string(body)
	require.Contains(t, s, `"success":false`)
	require.Contains(t, s, `"code":"TIMEOUT"`)
	require.Contains(t, s, "request timed out")
	require.NotContains(t, s, `"code":"INTERNAL"`)
}

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
