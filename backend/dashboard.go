package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

const dashboardLimit = persistence.MaxDashboardProducts

type dashboardStore interface {
	ListDashboardProducts(context.Context, int, bool) ([]domain.Product, error)
	CurrentProducts(context.Context, []string) (map[string][]persistence.CurrentListing, error)
	ProductsWithRecentAlerts(context.Context, []string, time.Time, time.Time) (map[string]bool, error)
}

type dashboardProduct struct {
	Product             productJSON    `json:"product"`
	ListingCount        int            `json:"listing_count"`
	HasCurrentPrice     bool           `json:"has_current_price"`
	BestPrice           *bestJSON      `json:"best_price"`
	ComparisonStatus    string         `json:"comparison_status"`
	LatestObservationAt *time.Time     `json:"latest_observation_at"`
	LatestAttemptAt     *time.Time     `json:"latest_attempt_at"`
	Freshness           map[string]int `json:"freshness"`
	Collection          map[string]int `json:"collection"`
	HasCollectionError  bool           `json:"has_collection_error"`
	RecentAlert         bool           `json:"recent_alert"`
}

// Reuse coherent valid observations and conservative comparison; no promotion
// or EffectivePrice data enters this read. Attempt times never affect freshness.
func summarizeProduct(p domain.Product, values []persistence.CurrentListing, status func(string) collector.Status, now time.Time, maxAge time.Duration) dashboardProduct {
	out := dashboardProduct{Product: productResponse(p), ListingCount: len(values), Freshness: map[string]int{}, Collection: map[string]int{}}
	items := make([]currentJSON, 0, len(values))
	for _, v := range values {
		state := status(v.Listing.ID)
		item := currentResponse(v, state, now, maxAge)
		state = item.Collection
		items = append(items, item)
		out.Freshness[item.Freshness]++
		out.Collection[state.State]++
		if state.State == "failed" || state.Error != "" {
			out.HasCollectionError = true
		}
		if state.LastAttemptedAt != nil && (out.LatestAttemptAt == nil || state.LastAttemptedAt.After(*out.LatestAttemptAt)) {
			at := *state.LastAttemptedAt
			out.LatestAttemptAt = &at
		}
		if v.Observation != nil {
			at := v.Observation.ObservedAt()
			if out.LatestObservationAt == nil || at.After(*out.LatestObservationAt) {
				out.LatestObservationAt = &at
			}
			if _, ok := domain.AlertPrice(*v.Observation); ok {
				out.HasCurrentPrice = true
			}
		}
	}
	out.BestPrice, out.ComparisonStatus = comparableBest(items)
	return out
}

func dashboardHandler(store dashboardStore, api currentAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		query := r.URL.Query()["include_archived"]
		if len(query) > 1 || (len(query) == 1 && query[0] != "true" && query[0] != "false") {
			productError(w, 400, "invalid_archive_filter", "include_archived must be true or false")
			return
		}
		includeArchived := len(query) == 1 && query[0] == "true"
		products, err := store.ListDashboardProducts(ctx, dashboardLimit+1, includeArchived)
		if err != nil {
			productError(w, 500, "internal_error", "unable to load dashboard")
			return
		}
		more := len(products) > dashboardLimit
		if more {
			products = products[:dashboardLimit]
		}
		ids := make([]string, 0, len(products))
		for _, p := range products {
			ids = append(ids, p.ID)
		}
		now := api.now()
		since := now.Add(-7 * 24 * time.Hour)
		alerts, err := store.ProductsWithRecentAlerts(ctx, ids, since, now)
		if err != nil {
			productError(w, 500, "internal_error", "unable to load dashboard alerts")
			return
		}
		values, err := store.CurrentProducts(ctx, ids)
		if err != nil {
			if errors.Is(err, persistence.ErrTooManyListings) {
				productError(w, 422, "too_many_listings", "dashboard supports at most 100 listings per product")
			} else {
				productError(w, 500, "internal_error", "unable to load dashboard prices")
			}
			return
		}
		data := make([]dashboardProduct, 0, len(products))
		for _, p := range products {
			summary := summarizeProduct(p, values[p.ID], api.status, now, api.maxAge)
			summary.RecentAlert = alerts[p.ID]
			data = append(data, summary)
		}
		writeJSON(w, 200, map[string]any{"data": map[string]any{"products": data, "truncated": more, "recent_alert_since": since}})
	}
}
