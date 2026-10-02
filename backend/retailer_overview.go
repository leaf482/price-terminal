package main

import (
	"context"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http"
	"time"
)

type retailerOverviewStore interface {
	GetRetailer(context.Context, string) (domain.Retailer, error)
	RetailerOverview(context.Context, string) ([]persistence.RetailerListing, bool, error)
}
type retailerListingJSON struct {
	Product productJSON `json:"product"`
	Current currentJSON `json:"current"`
}

func retailerOverviewHandler(store retailerOverviewStore, api currentAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		retailer, err := store.GetRetailer(ctx, r.PathValue("id"))
		if err != nil {
			catalogReadError(w, err, "retailer")
			return
		}
		rows, more, err := store.RetailerOverview(ctx, retailer.ID)
		if err != nil {
			productError(w, 500, "internal_error", "unable to load retailer overview")
			return
		}
		data := make([]retailerListingJSON, 0, len(rows))
		now := api.now()
		for _, row := range rows {
			data = append(data, retailerListingJSON{productResponse(row.Product), currentResponse(row.Current, api.status(row.Current.Listing.ID), now, api.maxAge)})
		}
		writeJSON(w, 200, map[string]any{"data": map[string]any{"retailer": retailerJSON{ID: retailer.ID, Name: retailer.Name}, "listings": data, "truncated": more}})
	}
}
