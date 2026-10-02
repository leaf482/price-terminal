//go:build integration

package persistence_test

import (
	"context"
	"github.com/leaf482/price-terminal/backend/domain"
	"testing"
	"time"
)

func TestRetailerOverviewIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := seedReliability(t, ctx, db)
	if _, err := s.InvalidateObservation(ctx, "bad", "exclude latest"); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertProduct(ctx, domain.Product{ID: "constructor", Name: "Second", Brand: "Maker", Model: "Model"}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertListing(ctx, domain.Listing{ID: "constructor", ProductID: "constructor", RetailerID: "r", URL: "https://example.com/second", TrackingDisabled: true}); err != nil {
		t.Fatal(err)
	}
	rows, more, err := s.RetailerOverview(ctx, "r")
	if err != nil || more || len(rows) != 2 {
		t.Fatal(rows, more, err)
	}
	if rows[0].Product.Name != "Second" || !rows[0].Current.Listing.TrackingDisabled || rows[0].Current.Observation != nil {
		t.Fatal(rows[0])
	}
	m, _ := rows[1].Current.Observation.OfferPrice()
	if m.MinorUnits != 100 {
		t.Fatal("invalidation filter lost", m)
	}
	o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "constructor", ObservedAt: time.Unix(3, 123456789), Source: "fixture", Stock: domain.StockUnknown, OfferPrice: &domain.Money{Currency: domain.JPY}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.InsertPriceObservation(ctx, "zero", o); err != nil {
		t.Fatal(err)
	}
	rows, _, err = s.RetailerOverview(ctx, "r")
	if err != nil {
		t.Fatal(err)
	}
	m, present := rows[0].Current.Observation.OfferPrice()
	if !present || m.MinorUnits != 0 || m.Currency != domain.JPY || rows[0].Current.Observation.ObservedAt().Nanosecond() != 123456789 {
		t.Fatal(rows[0])
	}
	empty, more, err := s.RetailerOverview(ctx, "empty")
	if err != nil || more || len(empty) != 0 {
		t.Fatal(empty, err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO listings(id,product_id,retailer_id,url,retailer_product_id) SELECT 'bulk-'||n,'constructor','r','https://example.com/'||n,'' FROM generate_series(1,99)n`)
	if err != nil {
		t.Fatal(err)
	}
	rows, more, err = s.RetailerOverview(ctx, "r")
	if err != nil || !more || len(rows) != 100 {
		t.Fatal(len(rows), more, err)
	}
}
