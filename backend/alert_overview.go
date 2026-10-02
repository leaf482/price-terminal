package main

import (
	"context"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http"
	"time"
)

type alertOverviewStore interface {
	AlertOverview(context.Context) (persistence.AlertOverview, error)
}

func alertOverviewHandler(store alertOverviewStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		data, err := store.AlertOverview(ctx)
		if err != nil {
			productError(w, 500, "internal_error", "unable to load alert overview")
			return
		}
		writeJSON(w, 200, map[string]any{"data": data})
	}
}
