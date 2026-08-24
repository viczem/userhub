package rest

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

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
					slog.Default().Error("close request body", "error", err)
				}
			}()

			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, int64(maxBodyBytes)))
			if err != nil {
				if maxBytesError, ok := errors.AsType[*http.MaxBytesError](err); ok {
					slog.Default().Error("max bytes", "error", maxBytesError)
					http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)

					return
				}

				slog.Default().Error("invalid request body", "error", err)
				http.Error(w, "invalid request body", http.StatusBadRequest)

				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}

func loggerMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			start := time.Now()

			defer func() {
				logger.InfoContext(r.Context(), r.URL.Path,
					slog.String("method", r.Method),
					slog.Int("status", ww.Status()),
					slog.Int("bytes", ww.BytesWritten()),
					slog.Int64("time", time.Since(start).Milliseconds()),
					slog.String("ip", r.RemoteAddr),
					slog.String("ua", r.UserAgent()),
					"tag", "rest",
				)
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}
