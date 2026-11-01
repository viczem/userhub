package rest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/viczem/userhub/services/userhub/internal/domain"
)

func maxBodyBytesMiddleware(maxBodyBytes int) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > int64(maxBodyBytes) {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}

			defer func() {
				if err := r.Body.Close(); err != nil {
					slog.WarnContext(r.Context(), "close request body", "tag", "rest")
				}
			}()

			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, int64(maxBodyBytes)))
			if err != nil {
				if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
					http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)

					return
				}

				slog.WarnContext(r.Context(), "invalid request body", "tag", "rest")
				http.Error(w, "invalid request body", http.StatusBadRequest)

				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		requestID := middleware.GetReqID(ctx)

		if requestID == "" {
			requestID = uuid.NewV4().String()
			ctx = context.WithValue(ctx, middleware.RequestIDKey, requestID)
		}

		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func loggerMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			var clientIPAttr slog.Attr
			if ip := middleware.GetClientIPAddr(r.Context()); ip.IsValid() {
				clientIPAttr = slog.String("ip", ip.String())
			}

			start := time.Now()

			defer func() {
				route := chi.RouteContext(r.Context()).RoutePattern()
				if route == "" {
					route = "unmatched"
				}

				slog.InfoContext(r.Context(), "http request",
					clientIPAttr,
					slog.String("method", r.Method),
					slog.String("route", route),
					slog.Int("status", ww.Status()),
					slog.Int("bytes", ww.BytesWritten()),
					slog.Int64("time", time.Since(start).Milliseconds()),
					"tag", "rest",
				)
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}

func recovererMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recover() == nil {
					return
				}

				slog.ErrorContext(r.Context(), "handler panic", "tag", "rest")
				writeJSON(w, r, http.StatusInternalServerError, nil)
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func validateConfigSessionMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-UserHub-Key")
			token = strings.TrimSpace(token)
			if token == "" {
				slog.ErrorContext(r.Context(), "missing token")
				writeJSON(w, r, http.StatusNotFound, nil)
				return
			}

			ctx := context.WithValue(r.Context(), domain.ConfigSessionTokenKey{}, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
