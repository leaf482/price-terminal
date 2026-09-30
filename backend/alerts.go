package main

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

type alertStore interface {
	GetListing(context.Context, string) (domain.Listing, error)
	InsertAlert(context.Context, domain.PriceAlert) error
	ListAlerts(context.Context, string) ([]domain.PriceAlert, error)
	SetAlertEnabled(context.Context, string, string, bool) (domain.PriceAlert, error)
	ListAlertEvents(context.Context, string) ([]persistence.AlertEvent, bool, error)
}
type alertAPI struct{ store alertStore }

func registerAlertRoutes(mux *http.ServeMux, store alertStore) {
	a := alertAPI{store}
	mux.HandleFunc("POST /listings/{id}/alerts", a.create)
	mux.HandleFunc("GET /listings/{id}/alerts", a.list)
	mux.HandleFunc("PATCH /listings/{id}/alerts/{alertID}", a.toggle)
	mux.HandleFunc("GET /listings/{id}/alert-events", a.events)
}
func (a alertAPI) create(w http.ResponseWriter, r *http.Request) {
	var body *struct {
		Kind      string          `json:"kind"`
		Currency  domain.Currency `json:"currency"`
		Threshold *int64          `json:"threshold_minor_units"`
		Drop      *int64          `json:"drop_basis_points"`
		Enabled   *bool           `json:"enabled"`
		Stock     *bool           `json:"require_in_stock"`
	}
	if err := decodeCatalogBody(w, r, &body); err != nil || body == nil {
		productError(w, 400, "invalid_request", "expected alert configuration")
		return
	}
	alert := domain.PriceAlert{ID: rand.Text(), ListingID: r.PathValue("id"), Kind: body.Kind, Currency: body.Currency, Threshold: body.Threshold, DropBasisPoints: body.Drop, Enabled: true, RequireInStock: true}
	if body.Enabled != nil {
		alert.Enabled = *body.Enabled
	}
	if body.Stock != nil {
		alert.RequireInStock = *body.Stock
	}
	if err := alert.Validate(); err != nil {
		productError(w, 400, "invalid_alert", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.store.InsertAlert(ctx, alert); err != nil {
		if errors.Is(err, persistence.ErrAlertLimit) {
			productError(w, 409, "alert_limit", "at most 100 alerts per listing")
		} else {
			catalogReadError(w, err, "listing")
		}
		return
	}
	writeJSON(w, 201, map[string]any{"data": alert})
}
func (a alertAPI) list(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	id := r.PathValue("id")
	if _, err := a.store.GetListing(ctx, id); err != nil {
		catalogReadError(w, err, "listing")
		return
	}
	alerts, err := a.store.ListAlerts(ctx, id)
	if err != nil {
		productError(w, 500, "internal_error", "unable to read alerts")
		return
	}
	writeJSON(w, 200, map[string]any{"data": alerts})
}
func (a alertAPI) toggle(w http.ResponseWriter, r *http.Request) {
	var body *struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decodeCatalogBody(w, r, &body); err != nil || body == nil || body.Enabled == nil {
		productError(w, 400, "invalid_request", "enabled must be an explicit boolean")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	alert, err := a.store.SetAlertEnabled(ctx, r.PathValue("id"), r.PathValue("alertID"), *body.Enabled)
	if err != nil {
		catalogReadError(w, err, "alert")
		return
	}
	writeJSON(w, 200, map[string]any{"data": alert})
}
func (a alertAPI) events(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	id := r.PathValue("id")
	if _, err := a.store.GetListing(ctx, id); err != nil {
		catalogReadError(w, err, "listing")
		return
	}
	events, more, err := a.store.ListAlertEvents(ctx, id)
	if err != nil {
		productError(w, 500, "internal_error", "unable to read alert events")
		return
	}
	writeJSON(w, 200, map[string]any{"data": map[string]any{"events": events, "truncated": more}})
}
