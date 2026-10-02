package main

import (
	"context"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http"
	"time"
)

type collectionHealthStore interface {
	CollectionHealth(context.Context) (persistence.CollectionHealth, error)
}

func collectionHealthHandler(s collectionHealthStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		result, err := s.CollectionHealth(ctx)
		if err != nil {
			productError(w, 500, "internal_error", "unable to load collection health")
			return
		}
		writeJSON(w, 200, map[string]any{"data": result})
	}
}
