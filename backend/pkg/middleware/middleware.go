package middleware

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"

	"aegis/pkg/response"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const requestIDHeader = "X-Request-ID"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r)
	})
}

func RequestLog(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			log.Info("http_request",
				zap.String("request_id", sw.Header().Get(requestIDHeader)),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", sw.status),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

// Timeout cancels the request context after d and, if the handler has not
// finished, writes a JSON 504. A late handler write cannot replace that body.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if d <= 0 {
				next.ServeHTTP(w, r)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()

			rec := &captureWriter{header: make(http.Header)}
			done := make(chan struct{})
			var panicVal any
			go func() {
				defer func() {
					panicVal = recover()
					close(done)
				}()
				next.ServeHTTP(rec, r.WithContext(ctx))
			}()

			var mu sync.Mutex
			finished := false
			select {
			case <-done:
				mu.Lock()
				defer mu.Unlock()
				if finished {
					return
				}
				finished = true
				if panicVal != nil {
					_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
					return
				}
				rec.flush(w)
			case <-ctx.Done():
				mu.Lock()
				defer mu.Unlock()
				if finished {
					return
				}
				finished = true
				_ = response.Error(w, http.StatusGatewayTimeout, "TIMEOUT", "request timed out")
			}
		})
	}
}

func Recoverer(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic_recovered", zap.Any("panic", rec))
					_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type captureWriter struct {
	header http.Header
	code   int
	buf    bytes.Buffer
	wrote  bool
}

func (c *captureWriter) Header() http.Header {
	return c.header
}

func (c *captureWriter) WriteHeader(code int) {
	if c.wrote {
		return
	}
	c.code = code
	c.wrote = true
}

func (c *captureWriter) Write(p []byte) (int, error) {
	if !c.wrote {
		c.WriteHeader(http.StatusOK)
	}
	return c.buf.Write(p)
}

func (c *captureWriter) flush(w http.ResponseWriter) {
	dst := w.Header()
	for k, vs := range c.header {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
	code := c.code
	if code == 0 {
		code = http.StatusOK
	}
	w.WriteHeader(code)
	_, _ = w.Write(c.buf.Bytes())
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
