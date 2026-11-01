package main

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5/middleware"
)

type requestIDHandler struct {
	next slog.Handler
}

var _ slog.Handler = (*requestIDHandler)(nil)

func newRequestIDHandler(next slog.Handler) *requestIDHandler {
	return &requestIDHandler{next: next}
}

func (h *requestIDHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *requestIDHandler) Handle(ctx context.Context, record slog.Record) error {
	if requestID := middleware.GetReqID(ctx); requestID != "" {
		record = record.Clone()
		record.AddAttrs(slog.String("id", requestID))
	}

	return h.next.Handle(ctx, record)
}

func (h *requestIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return newRequestIDHandler(h.next.WithAttrs(attrs))
}

func (h *requestIDHandler) WithGroup(name string) slog.Handler {
	return newRequestIDHandler(h.next.WithGroup(name))
}
