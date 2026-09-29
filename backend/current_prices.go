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

type currentStore interface {
	GetProduct(context.Context, string) (domain.Product, error)
	CurrentListing(context.Context, string) (persistence.CurrentListing, error)
	CurrentProduct(context.Context, string) ([]persistence.CurrentListing, error)
}

type observationJSON struct {
	ObservedAt        time.Time         `json:"observed_at"`
	Source            string            `json:"source"`
	Stock             domain.StockState `json:"stock"`
	Currency          domain.Currency   `json:"currency,omitempty"`
	MSRP              *int64            `json:"msrp,omitempty"`
	MSRPSource        string            `json:"msrp_source,omitempty"`
	RetailerListPrice *int64            `json:"retailer_list_price,omitempty"`
	SalePrice         *int64            `json:"sale_price,omitempty"`
	OfferPrice        *int64            `json:"offer_price,omitempty"`
}
type currentJSON struct {
	Listing     listingJSON      `json:"listing"`
	Observation *observationJSON `json:"observation"`
	Freshness   string           `json:"freshness"`
	Collection  collector.Status `json:"collection"`
}
type bestJSON struct {
	ListingID  string          `json:"listing_id"`
	RetailerID string          `json:"retailer_id"`
	MinorUnits int64           `json:"minor_units"`
	Currency   domain.Currency `json:"currency"`
	Basis      string          `json:"basis"`
}

func currentResponse(value persistence.CurrentListing, status collector.Status, now time.Time, maxAge time.Duration) currentJSON {
	result := currentJSON{Listing: listingResponse(value.Listing), Collection: status, Freshness: "missing"}
	if value.Observation == nil {
		return result
	}
	o := value.Observation
	result.Observation = observationResponse(*o)
	result.Freshness = "fresh"
	if o.ObservedAt().After(now) {
		result.Freshness = "future"
	} else if now.Sub(o.ObservedAt()) > maxAge {
		result.Freshness = "stale"
	}
	return result
}

func observationResponse(o domain.PriceObservation) *observationJSON {
	data := &observationJSON{ObservedAt: o.ObservedAt(), Source: o.Source(), Stock: o.Stock(), MSRPSource: o.MSRPSource()}
	data.Currency, _ = o.Currency()
	amount := func(m domain.Money, ok bool) *int64 {
		if !ok {
			return nil
		}
		return &m.MinorUnits
	}
	data.MSRP = amount(o.MSRP())
	data.RetailerListPrice = amount(o.RetailerListPrice())
	data.SalePrice = amount(o.SalePrice())
	data.OfferPrice = amount(o.OfferPrice())
	return data
}

// Conservative product-wide comparison: every listing must have a fresh,
// in-stock executable price in the same currency. Offer wins over sale; MSRP
// and retailer reference prices are never purchase-price fallbacks. Explicit
// zero is valid. Equal amounts select the bytewise smallest listing ID.
func comparableBest(items []currentJSON) (*bestJSON, string) {
	if len(items) == 0 {
		return nil, "no_listings"
	}
	var best *bestJSON
	for _, item := range items {
		o := item.Observation
		if o == nil {
			return nil, "missing_observation"
		}
		if item.Freshness != "fresh" {
			return nil, "not_fresh"
		}
		if o.Stock != domain.StockInStock {
			return nil, "not_in_stock"
		}
		price, basis := o.OfferPrice, "offer_price"
		if price == nil {
			price, basis = o.SalePrice, "sale_price"
		}
		if price == nil {
			return nil, "missing_price"
		}
		if best != nil && best.Currency != o.Currency {
			return nil, "incompatible_currencies"
		}
		if best == nil || *price < best.MinorUnits || (*price == best.MinorUnits && item.Listing.ID < best.ListingID) {
			best = &bestJSON{ListingID: item.Listing.ID, RetailerID: item.Listing.RetailerID, MinorUnits: *price, Currency: o.Currency, Basis: basis}
		}
	}
	return best, "comparable"
}

type currentAPI struct {
	store  currentStore
	status func(string) collector.Status
	maxAge time.Duration
	now    func() time.Time
}

func registerCurrentRoutes(mux *http.ServeMux, api currentAPI) {
	mux.HandleFunc("GET /listings/{id}/price", api.listing)
	mux.HandleFunc("GET /products/{id}/prices", api.product)
}
func (api currentAPI) listing(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	value, err := api.store.CurrentListing(ctx, r.PathValue("id"))
	if err != nil {
		catalogReadError(w, err, "listing")
		return
	}
	writeJSON(w, 200, map[string]any{"data": currentResponse(value, api.status(value.Listing.ID), api.now(), api.maxAge)})
}
func (api currentAPI) product(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	id := r.PathValue("id")
	if _, err := api.store.GetProduct(ctx, id); err != nil {
		catalogReadError(w, err, "product")
		return
	}
	values, err := api.store.CurrentProduct(ctx, id)
	if err != nil {
		if errors.Is(err, persistence.ErrTooManyListings) {
			productError(w, 422, "too_many_listings", "current-price comparison supports at most 100 listings")
		} else {
			productError(w, 500, "internal_error", "unable to read prices")
		}
		return
	}
	items := make([]currentJSON, 0, len(values))
	now := api.now()
	for _, value := range values {
		items = append(items, currentResponse(value, api.status(value.Listing.ID), now, api.maxAge))
	}
	best, reason := comparableBest(items)
	writeJSON(w, 200, map[string]any{"data": map[string]any{"product_id": id, "listings": items, "best_price": best, "comparison_status": reason}})
}
