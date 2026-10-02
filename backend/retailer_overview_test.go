package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http/httptest"
	"testing"
	"time"
)

type overviewStub struct {
	lookups, batches    int
	rows                []persistence.RetailerListing
	lookupErr, batchErr error
	more                bool
}

func (s *overviewStub) GetRetailer(_ context.Context, id string) (domain.Retailer, error) {
	s.lookups++
	return domain.Retailer{ID: id, Name: "Shop"}, s.lookupErr
}
func (s *overviewStub) RetailerOverview(context.Context, string) ([]persistence.RetailerListing, bool, error) {
	s.batches++
	return s.rows, s.more, s.batchErr
}
func TestRetailerOverview(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	old := now.Add(-time.Hour)
	o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "a", ObservedAt: old, Source: "fixture", Stock: domain.StockInStock, OfferPrice: &domain.Money{Currency: domain.USD}})
	if err != nil {
		t.Fatal(err)
	}
	rows := []persistence.RetailerListing{{Product: domain.Product{ID: "p", Name: "Product", Brand: "Brand", Model: "Model"}, Current: persistence.CurrentListing{Listing: domain.Listing{ID: "a", ProductID: "p", RetailerID: "constructor"}, Observation: &o}}, {Product: domain.Product{ID: "q"}, Current: persistence.CurrentListing{Listing: domain.Listing{ID: "b", ProductID: "q", RetailerID: "constructor", TrackingDisabled: true}}}}
	api := currentAPI{now: func() time.Time { return now }, maxAge: time.Minute, status: func(string) collector.Status {
		return collector.Status{State: "failed", Error: "collection_failed", LastAttemptedAt: &now}
	}}
	for _, tc := range []struct {
		name string
		s    overviewStub
		code int
	}{{"multiple", overviewStub{rows: rows, more: true}, 200}, {"empty", overviewStub{}, 200}, {"missing", overviewStub{lookupErr: sql.ErrNoRows}, 404}, {"failure", overviewStub{batchErr: errors.New("secret")}, 500}} {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.s
			r := httptest.NewRequest("GET", "/retailers/constructor/overview", nil)
			r.SetPathValue("id", "constructor")
			w := httptest.NewRecorder()
			retailerOverviewHandler(&s, api)(w, r)
			if w.Code != tc.code || s.lookups != 1 || s.batches > 1 {
				t.Fatal(w, s)
			}
			if tc.code == 200 {
				if s.batches != 1 {
					t.Fatal("batch not called")
				}
				var result struct {
					Data struct {
						Retailer  retailerJSON          `json:"retailer"`
						Listings  []retailerListingJSON `json:"listings"`
						Truncated bool                  `json:"truncated"`
					}
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Data.Retailer.ID != "constructor" || len(result.Data.Listings) != len(tc.s.rows) {
					t.Fatal(result)
				}
				if tc.name == "multiple" {
					a, b := result.Data.Listings[0], result.Data.Listings[1]
					if *a.Current.Observation.OfferPrice != 0 || a.Current.Freshness != "stale" || !a.Current.Observation.ObservedAt.Equal(old) || a.Current.Collection.State != "failed" || b.Current.Observation != nil || b.Current.Collection.State != "disabled" || !result.Data.Truncated {
						t.Fatal(result)
					}
				}
			}
		})
	}
}
