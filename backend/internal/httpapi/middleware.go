package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/rafael/estus-vault/backend/internal/domain"
	"github.com/rafael/estus-vault/backend/internal/service"
)

// withMiddleware wraps every request with request-id tagging, structured
// access logging, and panic recovery — the three things that are unsafe to
// skip on a service that will run unattended with no human watching a
// terminal. Authentication is not here: it applies to most of /api but not
// all of it, so it is mounted per route group in NewRouter instead.
func withMiddleware(next http.Handler) http.Handler {
	return recoverPanic(logRequests(next))
}

// ctxKey is unexported so nothing outside this package can write the
// authenticated user into a context — handlers can only read back what
// requireSession put there.
type ctxKey struct{}

var userCtxKey ctxKey

// UserFromContext returns the user requireSession authenticated for this
// request. The ok result is false on any route that isn't behind the
// middleware, so handlers must check it rather than assume a zero User means
// anything meaningful.
func UserFromContext(ctx context.Context) (domain.User, bool) {
	u, ok := ctx.Value(userCtxKey).(domain.User)
	return u, ok
}

// requireSession rejects any request without a live session token. Absent,
// malformed and expired tokens all produce the same bare 401: the client's
// only correct reaction to each is to send the user back to the login
// screen, so distinguishing them would only tell an attacker whether a token
// was ever real.
func requireSession(auth *service.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := auth.UserFromToken(r.Context(), bearerToken(r))
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Bearer realm="estus-brain"`)
			writeUnauthorized(w, errors.New("authentication required"))
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userCtxKey, user)))
	})
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "panic", rec, "path", r.URL.Path)
				writeError(w, errors.New("internal error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		slog.Info("request",
			"request_id", id,
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Flush passes streaming through the wrapper: the assistant's answers are
// server-sent events and must reach the browser as they are written.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
