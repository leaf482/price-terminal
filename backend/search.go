package main

import (
	"context"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type searchStore interface {
	SearchCatalog(context.Context, string) (persistence.CatalogSearch, error)
}

func searchHandler(store searchStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params, err := url.ParseQuery(r.URL.RawQuery)
		query := strings.TrimSpace(params.Get("q"))
		if err != nil || len(params["q"]) > 1 || persistence.ValidateSearch(query) != nil {
			productError(w, 400, "invalid_query", "q must be one text query of at most 200 characters without NUL")
			return
		}
		result := persistence.CatalogSearch{}
		if query != "" {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			result, err = store.SearchCatalog(ctx, query)
			if err != nil {
				productError(w, 500, "internal_error", "unable to search catalog")
				return
			}
		}
		products := make([]productJSON, 0, len(result.Products))
		retailers := make([]retailerJSON, 0, len(result.Retailers))
		listings := make([]listingJSON, 0, len(result.Listings))
		for _, p := range result.Products {
			products = append(products, productResponse(p))
		}
		for _, v := range result.Retailers {
			retailers = append(retailers, retailerJSON{ID: v.ID, Name: v.Name})
		}
		for _, l := range result.Listings {
			listings = append(listings, listingResponse(l))
		}
		writeJSON(w, 200, map[string]any{"data": map[string]any{"products": products, "retailers": retailers, "listings": listings}})
	}
}
