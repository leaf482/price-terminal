//go:build integration

package persistence_test

import (
	"context"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"testing"
	"time"
)

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
