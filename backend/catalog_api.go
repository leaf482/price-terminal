package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

type catalogStore interface {
	InsertRetailer(context.Context, domain.Retailer) error
	GetRetailer(context.Context, string) (domain.Retailer, error)
	ListRetailers(context.Context, int) ([]domain.Retailer, error)
	InsertListing(context.Context, domain.Listing) error
	GetListing(context.Context, string) (domain.Listing, error)
	ListListingsByProduct(context.Context, string, int) ([]domain.Listing, error)
	GetProduct(context.Context, string) (domain.Product, error)
}

type catalogAPI struct{ store catalogStore }

type retailerJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type listingJSON struct {
	ID                string `json:"id"`
	ProductID         string `json:"product_id"`
	RetailerID        string `json:"retailer_id"`
	URL               string `json:"url"`
	RetailerProductID string `json:"retailer_product_id"`
	TrackingEnabled   bool   `json:"tracking_enabled"`
}

func listingResponse(l domain.Listing) listingJSON {
	return listingJSON{ID: l.ID, ProductID: l.ProductID, RetailerID: l.RetailerID, URL: l.URL, RetailerProductID: l.RetailerProductID, TrackingEnabled: l.TrackingEnabled()}
}

func registerCatalogRoutes(mux *http.ServeMux, store catalogStore) {
	api := catalogAPI{store: store}
	mux.HandleFunc("POST /retailers", api.createRetailer)
	mux.HandleFunc("GET /retailers/{id}", api.getRetailer)
	mux.HandleFunc("GET /retailers", api.listRetailers)
	mux.HandleFunc("POST /listings", api.createListing)
	mux.HandleFunc("GET /listings/{id}", api.getListing)
	mux.HandleFunc("GET /products/{productID}/listings", api.listListings)
}

func catalogLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	limit, ok := listLimit(query["limit"], persistence.MaxCatalogListLimit)
	if err != nil || !ok {
		productError(w, 400, "invalid_limit", "limit must be an integer from 1 to 100")
		return 0, false
	}
	return limit, true
}

func catalogReadError(w http.ResponseWriter, err error, resource string) {
	if errors.Is(err, sql.ErrNoRows) {
		productError(w, 404, resource+"_not_found", resource+" not found")
	} else {
		productError(w, 500, "internal_error", "unable to read "+resource)
	}
}

func (api catalogAPI) createRetailer(w http.ResponseWriter, r *http.Request) {
	var body *retailerJSON
	if err := decodeCatalogBody(w, r, &body); err != nil || body == nil {
		productError(w, 400, "invalid_request", "expected a retailer JSON object")
		return
	}
	retailer := domain.Retailer{ID: body.ID, Name: body.Name}
	if err := retailer.Validate(); err != nil {
		productError(w, 400, "invalid_retailer", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := api.store.InsertRetailer(ctx, retailer); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == "23505" && pgError.ConstraintName == "retailers_pkey" {
			productError(w, 409, "duplicate_retailer", "retailer ID already exists")
		} else {
			productError(w, 500, "internal_error", "unable to create retailer")
		}
		return
	}
	writeJSON(w, 201, map[string]any{"data": body})
}

func (api catalogAPI) getRetailer(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	retailer, err := api.store.GetRetailer(ctx, r.PathValue("id"))
	if err != nil {
		catalogReadError(w, err, "retailer")
		return
	}
	writeJSON(w, 200, map[string]any{"data": retailerJSON{ID: retailer.ID, Name: retailer.Name}})
}

func (api catalogAPI) listRetailers(w http.ResponseWriter, r *http.Request) {
	limit, ok := catalogLimit(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	retailers, err := api.store.ListRetailers(ctx, limit)
	if err != nil {
		productError(w, 500, "internal_error", "unable to list retailers")
		return
	}
	data := make([]retailerJSON, 0, len(retailers))
	for _, retailer := range retailers {
		data = append(data, retailerJSON{ID: retailer.ID, Name: retailer.Name})
	}
	writeJSON(w, 200, map[string]any{"data": data})
}

func (api catalogAPI) createListing(w http.ResponseWriter, r *http.Request) {
	var body *struct {
		ID                string `json:"id"`
		ProductID         string `json:"product_id"`
		RetailerID        string `json:"retailer_id"`
		URL               string `json:"url"`
		RetailerProductID string `json:"retailer_product_id"`
		TrackingEnabled   *bool  `json:"tracking_enabled"`
	}
	if err := decodeCatalogBody(w, r, &body); err != nil || body == nil {
		productError(w, 400, "invalid_request", "expected a listing JSON object")
		return
	}
	listing := domain.Listing{ID: body.ID, ProductID: body.ProductID, RetailerID: body.RetailerID, URL: body.URL, RetailerProductID: body.RetailerProductID}
	if body.TrackingEnabled != nil {
		listing.TrackingDisabled = !*body.TrackingEnabled
	}
	if err := listing.Validate(); err != nil {
		productError(w, 400, "invalid_listing", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	// The INSERT's foreign keys enforce existence atomically, without a racy
	// precheck or automatically creating related records.
	if err := api.store.InsertListing(ctx, listing); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) {
			switch {
			case pgError.Code == "23505" && pgError.ConstraintName == "listings_pkey":
				productError(w, 409, "duplicate_listing", "listing ID already exists")
				return
			case pgError.Code == "23505" && pgError.ConstraintName == "listings_source_identity_key":
				productError(w, 409, "duplicate_listing_source", "listing source identity already exists")
				return
			case pgError.Code == "23503" && pgError.ConstraintName == "listings_product_id_fkey":
				productError(w, 400, "invalid_product_reference", "referenced product does not exist")
				return
			case pgError.Code == "23503" && pgError.ConstraintName == "listings_retailer_id_fkey":
				productError(w, 400, "invalid_retailer_reference", "referenced retailer does not exist")
				return
			}
		}
		productError(w, 500, "internal_error", "unable to create listing")
		return
	}
	writeJSON(w, 201, map[string]any{"data": listingResponse(listing)})
}

func (api catalogAPI) getListing(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	listing, err := api.store.GetListing(ctx, r.PathValue("id"))
	if err != nil {
		catalogReadError(w, err, "listing")
		return
	}
	writeJSON(w, 200, map[string]any{"data": listingResponse(listing)})
}

func (api catalogAPI) listListings(w http.ResponseWriter, r *http.Request) {
	limit, ok := catalogLimit(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	productID := r.PathValue("productID")
	if _, err := api.store.GetProduct(ctx, productID); err != nil {
		catalogReadError(w, err, "product")
		return
	}
	listings, err := api.store.ListListingsByProduct(ctx, productID, limit)
	if err != nil {
		productError(w, 500, "internal_error", "unable to list listings")
		return
	}
	data := make([]listingJSON, 0, len(listings))
	for _, listing := range listings {
		data = append(data, listingResponse(listing))
	}
	writeJSON(w, 200, map[string]any{"data": data})
}
