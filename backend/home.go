package main

import (
	"context"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http"
	"time"
)

type homeStore interface {
	HomeCounts(context.Context, time.Time) (persistence.HomeCounts, error)
	RecentPriceChanges(context.Context) (persistence.PriceChanges, error)
	AlertOverview(context.Context) (persistence.AlertOverview, error)
	RecentCollectionFailures(context.Context) ([]persistence.HomeFailure, error)
}

// Fixed query count, no per-row reads. Sections fail independently. Existing
// feeds retain their ordering/semantics; only their first five items are exposed.
func homeHandler(s homeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		errors := []string{}
		data := map[string]any{"counts": nil, "price_changes": nil, "alert_events": nil, "collection_failures": nil}
		run := func(name string, read func(context.Context) (any, error)) {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			v, err := read(ctx)
			if err != nil {
				errors = append(errors, name)
			} else {
				data[name] = v
			}
		}
		run("counts", func(ctx context.Context) (any, error) { return s.HomeCounts(ctx, time.Now().UTC()) })
		run("price_changes", func(ctx context.Context) (any, error) {
			v, err := s.RecentPriceChanges(ctx)
			return v.Changes[:min(5, len(v.Changes))], err
		})
		run("alert_events", func(ctx context.Context) (any, error) {
			v, err := s.AlertOverview(ctx)
			return v.Events[:min(5, len(v.Events))], err
		})
		run("collection_failures", func(ctx context.Context) (any, error) { return s.RecentCollectionFailures(ctx) })
		data["errors"] = errors
		writeJSON(w, 200, map[string]any{"data": data})
	}
}
