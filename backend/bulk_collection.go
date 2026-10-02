package main

import (
	"context"
	"github.com/leaf482/price-terminal/backend/collector"
	"net/http"
)

func bulkCollectionHandler(collect func(context.Context, []string) ([]collector.BulkResult, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body *struct {
			ListingIDs []string `json:"listing_ids"`
		}
		if err := decodeCatalogBody(w, r, &body); err != nil || body == nil {
			productError(w, 400, "invalid_batch", "Provide listing_ids with 1..20 nonblank IDs.")
			return
		}
		ids, err := collector.BulkIDs(body.ListingIDs)
		if err != nil {
			productError(w, 400, "invalid_batch", err.Error())
			return
		}
		run := collect
		if run == nil {
			var runtime *collector.Runtime
			run = runtime.CollectBulk
		}
		results, err := run(r.Context(), ids)
		if err != nil {
			productError(w, 500, "internal_error", "Unable to collect Listings.")
			return
		}
		// A valid batch returns per-item outcomes even with partial failure/cancellation.
		writeJSON(w, 200, map[string]any{"data": results})
	}
}
