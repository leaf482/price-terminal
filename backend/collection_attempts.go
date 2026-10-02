package main

import (
	"context"
	"github.com/leaf482/price-terminal/backend/domain"
	"net/http"
	"time"
)

type attemptStore interface {
	GetListing(context.Context, string) (domain.Listing, error)
	ListCollectionAttempts(context.Context, string) ([]domain.CollectionAttempt, error)
}

func registerAttemptRoutes(mux *http.ServeMux, s attemptStore) {
	mux.HandleFunc("GET /listings/{id}/collection-attempts", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		id := r.PathValue("id")
		if _, err := s.GetListing(ctx, id); err != nil {
			catalogReadError(w, err, "listing")
			return
		}
		items, err := s.ListCollectionAttempts(ctx, id)
		if err != nil {
			productError(w, 500, "internal_error", "unable to read collection attempts")
			return
		}
		writeJSON(w, 200, map[string]any{"data": items})
	})
}
