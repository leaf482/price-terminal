package main

import (
	"context"
	"net/http"
	"time"
)

type trackingStore interface {
	SetListingTracking(context.Context, string, bool) error
}

func trackingHandler(store trackingStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body *struct {
			Enabled *bool `json:"tracking_enabled"`
		}
		if err := decodeCatalogBody(w, r, &body); err != nil || body == nil || body.Enabled == nil {
			productError(w, 400, "invalid_tracking", "tracking_enabled must be an explicit boolean")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := store.SetListingTracking(ctx, r.PathValue("id"), *body.Enabled); err != nil {
			catalogReadError(w, err, "listing")
			return
		}
		writeJSON(w, 200, map[string]any{"data": map[string]any{"listing_id": r.PathValue("id"), "tracking_enabled": *body.Enabled}})
	}
}
