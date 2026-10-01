package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

func TestDashboardSummary(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	makeRow := func(id string, money *domain.Money, at time.Time) persistence.CurrentListing {
		o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: id, Source: "manual: evidence", Stock: domain.StockInStock, ObservedAt: at, OfferPrice: money})
		if err != nil {
			t.Fatal(err)
		}
		return persistence.CurrentListing{Listing: domain.Listing{ID: id, ProductID: "p", RetailerID: "r", URL: "https://example.com/" + id}, Observation: &o}
	}
	zero := makeRow("a", &domain.Money{MinorUnits: 0, Currency: domain.USD}, now.Add(-time.Minute))
	missing := persistence.CurrentListing{Listing: domain.Listing{ID: "b", ProductID: "p", RetailerID: "s", URL: "https://example.com/b"}}
	status := func(string) collector.Status {
		return collector.Status{State: "failed", LastAttemptedAt: &now, Error: "collection_failed"}
	}
	for _, tc := range []struct {
		name           string
		rows           []persistence.CurrentListing
		hasPrice, best bool
		reason         string
	}{
		{"zero", []persistence.CurrentListing{zero}, true, true, "comparable"},
		{"no listings", nil, false, false, "no_listings"},
		{"missing", []persistence.CurrentListing{missing}, false, false, "missing_observation"},
		{"partial", []persistence.CurrentListing{zero, missing}, true, false, "missing_observation"},
		{"stock only", []persistence.CurrentListing{makeRow("b", nil, now)}, false, false, "missing_price"},
		{"mixed currencies", []persistence.CurrentListing{zero, makeRow("b", &domain.Money{MinorUnits: 10, Currency: domain.JPY}, now)}, true, false, "incompatible_currencies"},
		{"stale", []persistence.CurrentListing{makeRow("b", &domain.Money{MinorUnits: 10, Currency: domain.USD}, now.Add(-time.Hour))}, true, false, "not_fresh"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := summarizeProduct(domain.Product{ID: "p", Name: "Item", Brand: "Brand", Model: "Model"}, tc.rows, status, now, 15*time.Minute)
			if got.ListingCount != len(tc.rows) || got.HasCurrentPrice != tc.hasPrice || (got.BestPrice != nil) != tc.best || got.ComparisonStatus != tc.reason {
				t.Fatalf("%+v", got)
			}
			if tc.name == "zero" && (got.BestPrice.MinorUnits != 0 || !got.LatestObservationAt.Equal(zero.Observation.ObservedAt()) || !got.LatestAttemptAt.Equal(now) || got.Freshness["fresh"] != 1 || !got.HasCollectionError) {
				t.Fatal("zero/time/status semantics", got)
			}
			if tc.name == "stale" && (got.Freshness["stale"] != 1 || got.LatestObservationAt.Equal(*got.LatestAttemptAt)) {
				t.Fatal("failure refreshed old facts", got)
			}
		})
	}
}

type dashboardStub struct {
	includeArchived         bool
	products                []domain.Product
	err, priceErr, alertErr error
	limit, calls            int
	ids                     []string
	priceIDs                []string
	since, until            time.Time
}

func (s *dashboardStub) ListDashboardProducts(_ context.Context, limit int, includeArchived bool) ([]domain.Product, error) {
	s.includeArchived = includeArchived
	s.limit = limit
	return s.products, s.err
}
func (s *dashboardStub) CurrentProducts(_ context.Context, ids []string) (map[string][]persistence.CurrentListing, error) {
	s.calls++
	s.priceIDs = append([]string{}, ids...)
	return nil, s.priceErr
}
func (s *dashboardStub) ProductsWithRecentAlerts(_ context.Context, ids []string, since, until time.Time) (map[string]bool, error) {
	s.ids = ids
	s.since = since
	s.until = until
	return map[string]bool{"p00": true}, s.alertErr
}
func TestDashboardEndpoint(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	api := currentAPI{now: func() time.Time { return now }, status: func(string) collector.Status { return collector.Status{State: "inactive"} }, maxAge: time.Minute}
	for _, tc := range []struct {
		name                    string
		n                       int
		err, priceErr, alertErr error
		want                    int
	}{
		{name: "empty", want: 200}, {name: "bounded", n: 21, want: 200},
		{name: "products fail", err: errors.New("secret"), want: 500},
		{name: "prices fail", n: 1, priceErr: errors.New("secret"), want: 500},
		{name: "oversized product", n: 1, priceErr: persistence.ErrTooManyListings, want: 422},
		{name: "alerts fail", n: 1, alertErr: errors.New("secret"), want: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &dashboardStub{err: tc.err, priceErr: tc.priceErr, alertErr: tc.alertErr}
			for i := 0; i < tc.n; i++ {
				s.products = append(s.products, domain.Product{ID: fmt.Sprintf("p%02d", i)})
			}
			w := httptest.NewRecorder()
			dashboardHandler(s, api)(w, httptest.NewRequest("GET", "/dashboard", nil))
			if w.Code != tc.want {
				t.Fatal(w.Code, w.Body)
			}
			if tc.want != 200 {
				return
			}
			var response struct {
				Data struct {
					Products  []dashboardProduct `json:"products"`
					Truncated bool               `json:"truncated"`
				}
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if s.limit != 21 || s.calls != 1 || len(s.priceIDs) != min(tc.n, 20) || len(s.ids) > 20 || len(response.Data.Products) != min(tc.n, 20) || response.Data.Truncated != (tc.n > 20) {
				t.Fatal("bound", s, response)
			}
			if !s.since.Equal(now.Add(-7*24*time.Hour)) || !s.until.Equal(now) {
				t.Fatal("alert window")
			}
			if tc.n > 0 && !response.Data.Products[0].RecentAlert {
				t.Fatal("missing event indicator")
			}
		})
	}
}
