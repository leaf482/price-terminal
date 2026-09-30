package main

import (
	"context"
	"errors"
	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"net/http"
	"time"
)

func (api currentAPI) collectListing(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	value, err := api.store.CurrentListing(ctx, id)
	cancel()
	if err != nil {
		catalogReadError(w, err, "listing")
		return
	}
	if value.Listing.TrackingDisabled {
		productError(w, 409, "tracking_disabled", "Tracking is disabled for this Listing.")
		return
	}
	if api.collect == nil {
		productError(w, 503, "collection_unavailable", "No provider configured for this Listing.")
		return
	}
	err = api.collect(r.Context(), id)
	switch {
	case errors.Is(err, domain.ErrTrackingDisabled):
		productError(w, 409, "tracking_disabled", "Tracking is disabled for this Listing.")
	case errors.Is(err, collector.ErrUnavailable):
		productError(w, 503, "collection_unavailable", "No provider configured for this Listing.")
	case errors.Is(err, collector.ErrBusy):
		productError(w, 409, "collection_busy", "Collection is already running for this Listing.")
	case err != nil:
		productError(w, 502, "collection_failed", "Collection failed. Previous observations are preserved.")
	default:
		writeJSON(w, 200, map[string]any{"data": map[string]any{"listing_id": id, "collection": api.status(id)}})
	}
}
