package rest

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

func loggerMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			start := time.Now()

			defer func() {
				route := chi.RouteContext(r.Context()).RoutePattern()
				if route == "" {
					route = "unmatched"
				}

				slog.InfoContext(r.Context(), "http request",
					slog.String("method", r.Method),
					slog.String("route", route),
					slog.Int("status", ww.Status()),
					slog.Int("bytes", ww.BytesWritten()),
					slog.Int64("time", time.Since(start).Milliseconds()),
					slog.String("request_id", middleware.GetReqID(r.Context())),
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
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}()

			next.ServeHTTP(w, r)
		})
	}
}
