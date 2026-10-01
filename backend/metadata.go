package main

import (
	"context"
	"github.com/leaf482/price-terminal/backend/domain"
	"net/http"
	"time"
)

type metadataStore interface {
	UpdateProductMetadata(context.Context, domain.Product) (domain.Product, error)
	UpdateRetailerMetadata(context.Context, domain.Retailer) (domain.Retailer, error)
}

func registerMetadataRoutes(mux *http.ServeMux, store metadataStore) {
	mux.HandleFunc("PUT /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		var body *struct {
			Name  *string `json:"name"`
			Brand *string `json:"brand"`
			Model *string `json:"model"`
		}
		if err := decodeCatalogBody(w, r, &body); err != nil || body == nil || body.Name == nil || body.Brand == nil || body.Model == nil {
			productError(w, 400, "invalid_request", "name, brand and model must be explicit strings; IDs are immutable")
			return
		}
		p := domain.Product{ID: r.PathValue("id"), Name: *body.Name, Brand: *body.Brand, Model: *body.Model}
		if err := p.Validate(); err != nil {
			productError(w, 400, "invalid_product", err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		saved, err := store.UpdateProductMetadata(ctx, p)
		if err != nil {
			catalogReadError(w, err, "product")
			return
		}
		writeJSON(w, 200, map[string]any{"data": productResponse(saved)})
	})
	mux.HandleFunc("PUT /retailers/{id}", func(w http.ResponseWriter, r *http.Request) {
		var body *struct {
			Name *string `json:"name"`
		}
		if err := decodeCatalogBody(w, r, &body); err != nil || body == nil || body.Name == nil {
			productError(w, 400, "invalid_request", "name must be an explicit string; IDs are immutable")
			return
		}
		retailer := domain.Retailer{ID: r.PathValue("id"), Name: *body.Name}
		if err := retailer.Validate(); err != nil {
			productError(w, 400, "invalid_retailer", err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		saved, err := store.UpdateRetailerMetadata(ctx, retailer)
		if err != nil {
			catalogReadError(w, err, "retailer")
			return
		}
		writeJSON(w, 200, map[string]any{"data": retailerJSON{ID: saved.ID, Name: saved.Name}})
	})
}
