//go:build integration

package persistence_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"reflect"
	"testing"
	"time"
)

func TestCurrentProductsBatchIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s := persistence.New(freshDatabase(t, ctx))
	if err := s.InsertRetailer(ctx, domain.Retailer{ID: "r"}); err != nil {
		t.Fatal(err)
	}
	ids := []string{"usd", "jpy", "empty", "unobserved", "stock", "limit"}
	for _, id := range ids {
		if err := s.InsertProduct(ctx, domain.Product{ID: id}); err != nil {
			t.Fatal(err)
		}
	}
	listing := func(id, product string) {
		t.Helper()
		if err := s.InsertListing(ctx, domain.Listing{ID: id, ProductID: product, RetailerID: "r", URL: "https://example.com/" + id}); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"usd", "jpy", "unobserved", "stock"} {
		listing(id, id)
	}
	// An additional Listing verifies deterministic per-Product ordering.
	listing("aaa", "usd")
	at := time.Date(2026, 9, 30, 12, 0, 0, 123456789, time.UTC)
	insert := func(id, l string, when time.Time, offer, sale *domain.Money) {
		t.Helper()
		o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: l, ObservedAt: when, Source: id, Stock: domain.StockInStock, OfferPrice: offer, SalePrice: sale})
		if err != nil {
			t.Fatal(err)
		}
		if err := s.InsertPriceObservation(ctx, id, o); err != nil {
			t.Fatal(err)
		}
	}
	usd := &domain.Money{MinorUnits: 0, Currency: domain.USD}
	jpy := &domain.Money{MinorUnits: 10, Currency: domain.JPY}
	insert("a", "usd", at, &domain.Money{MinorUnits: 99, Currency: domain.USD}, nil)
	insert("z", "usd", at, usd, &domain.Money{MinorUnits: 50, Currency: domain.USD})
	insert("invalid", "usd", at.Add(time.Minute), &domain.Money{MinorUnits: 777, Currency: domain.USD}, nil)
	if _, err := s.InvalidateObservation(ctx, "invalid", "bad observation"); err != nil {
		t.Fatal(err)
	}
	insert("jpy-sale", "jpy", at, nil, jpy)
	insert("stock-only", "stock", at, nil, nil)
	for i := 0; i < persistence.MaxCurrentListings; i++ {
		listing(fmt.Sprintf("limit%03d", i), "limit")
	}
	got, err := s.CurrentProducts(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	// Batch and existing single reads must agree on every field and ordering.
	for _, id := range ids {
		single, err := s.CurrentProduct(ctx, id)
		if err != nil || !reflect.DeepEqual(got[id], single) {
			t.Fatalf("product %s mismatch: %v", id, err)
		}
	}
	if len(got["empty"]) != 0 || got["unobserved"][0].Observation != nil {
		t.Fatal("missing observations invented")
	}
	u := got["usd"][1].Observation
	if got["usd"][0].Listing.ID != "aaa" || u.Source() != "z" || !u.ObservedAt().Equal(at) {
		t.Fatal("invalidation/order/tie precision changed")
	}
	if m, ok := domain.AlertPrice(*u); !ok || m != *usd {
		t.Fatal("explicit zero or offer precedence lost")
	}
	if m, ok := domain.AlertPrice(*got["jpy"][0].Observation); !ok || m != *jpy {
		t.Fatal("currency/sale fallback lost")
	}
	if _, ok := got["stock"][0].Observation.Currency(); ok {
		t.Fatal("stock-only currency invented")
	}
	// Only the requested Products can overflow; other Products remain usable.
	listing("limit-overflow", "limit")
	if _, err := s.CurrentProducts(ctx, []string{"usd", "limit"}); !errors.Is(err, persistence.ErrTooManyListings) {
		t.Fatal("per-product limit lost", err)
	}
	if _, err := s.CurrentProducts(ctx, []string{"usd", "jpy"}); err != nil {
		t.Fatal("unrequested overflow affected batch", err)
	}
	duplicate, err := s.CurrentProducts(ctx, []string{"usd", "usd"})
	if err != nil || len(duplicate["usd"]) != 2 {
		t.Fatal("duplicate IDs duplicated listings", err)
	}
}

func TestDashboardRecentAlertsIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := persistence.New(db)
	if err := s.InsertRetailer(ctx, domain.Retailer{ID: "r"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	since := now.Add(-7 * 24 * time.Hour)
	for _, entry := range []struct {
		id string
		at time.Time
	}{{"recent", now}, {"boundary", since}, {"old", since.Add(-time.Microsecond)}, {"future", now.Add(time.Minute)}} {
		id := entry.id
		if err := s.InsertProduct(ctx, domain.Product{ID: id}); err != nil {
			t.Fatal(err)
		}
		if err := s.InsertListing(ctx, domain.Listing{ID: id, ProductID: id, RetailerID: "r", URL: "https://example.com/" + id}); err != nil {
			t.Fatal(err)
		}
		threshold := int64(1)
		if err := s.InsertAlert(ctx, domain.PriceAlert{ID: id, ListingID: id, Kind: "target", Currency: domain.USD, Threshold: &threshold, Enabled: true}); err != nil {
			t.Fatal(err)
		}
		o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: id, ObservedAt: now.Add(-time.Hour), Source: "manual: evidence", Stock: domain.StockInStock, OfferPrice: &domain.Money{MinorUnits: 0, Currency: domain.USD}})
		if err != nil {
			t.Fatal(err)
		}
		if err := s.InsertPriceObservation(ctx, id, o); err != nil {
			t.Fatal(err)
		}
		if err := s.EvaluateAlerts(ctx, id); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `UPDATE price_alert_events SET triggered_at=$1 WHERE alert_id=$2`, entry.at, id); err != nil {
			t.Fatal(err)
		}
	}
	// Invalidation excludes current facts but must not rewrite the event indicator.
	if _, err := s.InvalidateObservation(ctx, "recent", "incorrect evidence"); err != nil {
		t.Fatal(err)
	}
	got, err := s.ProductsWithRecentAlerts(ctx, []string{"recent", "boundary", "old", "future"}, since, now)
	if err != nil || len(got) != 2 || !got["recent"] || !got["boundary"] {
		t.Fatal(got, err)
	}
	subset, err := s.ProductsWithRecentAlerts(ctx, []string{"old"}, since, now)
	if err != nil || len(subset) != 0 {
		t.Fatal("subset", subset, err)
	}
	empty, err := s.ProductsWithRecentAlerts(ctx, nil, since, now)
	if err != nil || len(empty) != 0 {
		t.Fatal(empty, err)
	}
}
