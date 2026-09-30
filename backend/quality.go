package main

import (
	"context"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type qualityStore interface {
	GetListing(context.Context, string) (domain.Listing, error)
	ListObservationAudit(context.Context, string) ([]persistence.ObservationAudit, bool, error)
	GetObservationAudit(context.Context, string) (persistence.ObservationAudit, error)
	InvalidateObservation(context.Context, string, string) (persistence.ObservationAudit, error)
}

func auditResponse(a persistence.ObservationAudit) map[string]any {
	return map[string]any{"id": a.ID, "observation": observationResponse(a.Observation), "listing_id": a.Observation.ListingID(), "valid": a.InvalidatedAt == nil, "invalidation_reason": a.Reason, "invalidated_at": a.InvalidatedAt}
}
func registerQualityRoutes(mux *http.ServeMux, s qualityStore) {
	mux.HandleFunc("GET /listings/{id}/observations", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		id := r.PathValue("id")
		if _, err := s.GetListing(ctx, id); err != nil {
			catalogReadError(w, err, "listing")
			return
		}
		rows, more, err := s.ListObservationAudit(ctx, id)
		if err != nil {
			productError(w, 500, "internal_error", "unable to read observation audit")
			return
		}
		data := make([]map[string]any, 0, len(rows))
		for _, a := range rows {
			data = append(data, auditResponse(a))
		}
		writeJSON(w, 200, map[string]any{"data": map[string]any{"observations": data, "truncated": more}})
	})
	mux.HandleFunc("GET /observations/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		a, err := s.GetObservationAudit(ctx, r.PathValue("id"))
		if err != nil {
			catalogReadError(w, err, "observation")
			return
		}
		writeJSON(w, 200, map[string]any{"data": auditResponse(a)})
	})
	mux.HandleFunc("POST /observations/{id}/invalidate", func(w http.ResponseWriter, r *http.Request) {
		var body *struct {
			Reason string `json:"reason"`
		}
		if err := decodeCatalogBody(w, r, &body); err != nil || body == nil || strings.TrimSpace(body.Reason) == "" || utf8.RuneCountInString(body.Reason) > 1000 {
			productError(w, 400, "invalid_invalidation", "reason must contain 1..1000 characters")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		a, err := s.InvalidateObservation(ctx, r.PathValue("id"), body.Reason)
		if err != nil {
			if errors.Is(err, persistence.ErrInvalidInvalidation) {
				productError(w, 400, "invalid_invalidation", err.Error())
			} else {
				catalogReadError(w, err, "observation")
			}
			return
		}
		slog.Info("observation_invalidation", "observation_id", a.ID, "listing_id", a.Observation.ListingID(), "invalidated_at", a.InvalidatedAt)
		writeJSON(w, 200, map[string]any{"data": auditResponse(a)})
	})
}
