package httpapi

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/example/ride-matching/internal/observability"
)

type contextKey string

const requestIDKey contextKey = "request-id"

func (s *Server) registerMiddleware() {
	s.mux.Use(s.recoverMiddleware)
	s.mux.Use(s.requestIDMiddleware)
	s.mux.Use(s.observabilityMiddleware)
}

func (s *Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = newID()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) observabilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)

		route := routeTemplate(r)
		status := strconv.Itoa(ww.status)

		observability.HTTPRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
		observability.HTTPRequestDuration.WithLabelValues(r.Method, route, status).Observe(time.Since(start).Seconds())

		args := []any{
			"method", r.Method,
			"route", route,
			"status", ww.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", remoteIP(r),
		}
		if rid := requestIDFromContext(r.Context()); rid != "" {
			args = append(args, "request_id", rid)
		}
		s.logger.Info("http_request", args...)
	})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error("panic recovered", "error", rec)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (r *responseWriter) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func requestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func routeTemplate(r *http.Request) string {
	if current := mux.CurrentRoute(r); current != nil {
		if tmpl, err := current.GetPathTemplate(); err == nil {
			return tmpl
		}
	}
	return r.URL.Path
}

func remoteIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
