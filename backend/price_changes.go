package main

import (
	"context"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http"
	"time"
)

type priceChangesStore interface {
	RecentPriceChanges(context.Context) (persistence.PriceChanges, error)
}

func priceChangesHandler(s priceChangesStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		data, err := s.RecentPriceChanges(ctx)
		if err != nil {
			productError(w, 500, "internal_error", "unable to load price changes")
			return
		}
		writeJSON(w, 200, map[string]any{"data": data})
	}
}
