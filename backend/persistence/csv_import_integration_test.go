//go:build integration

package persistence_test

import (
	"context"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"testing"
	"time"
)

func TestCSVImportTransactionIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s := persistence.New(freshDatabase(t, ctx))
	for _, err := range []error{s.InsertProduct(ctx, domain.Product{ID: "p"}), s.InsertRetailer(ctx, domain.Retailer{ID: "r"}), s.InsertListing(ctx, domain.Listing{ID: "l", ProductID: "p", RetailerID: "r", URL: "https://example.com/l", TrackingDisabled: true})} {
		if err != nil {
			t.Fatal(err)
		}
	}
	threshold := int64(100)
	if err := s.InsertAlert(ctx, domain.PriceAlert{ID: "a", ListingID: "l", Kind: "target", Currency: domain.USD, Threshold: &threshold, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 1, 2, 3, 4, 5, 123456789, time.UTC)
	makeRow := func(when time.Time) domain.PriceObservation {
		t.Helper()
		o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: when, Source: "manual: CSV import", Stock: domain.StockInStock, OfferPrice: &domain.Money{MinorUnits: 0, Currency: domain.USD}})
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	rows := []domain.PriceObservation{makeRow(at), makeRow(at.Add(time.Hour))}
	if err := s.ImportObservations(ctx, "l", "good", rows); err != nil {
		t.Fatal(err)
	}
	history, err := s.PriceHistory(ctx, "l", nil, at.Add(2*time.Hour))
	if err != nil || len(history.Observations) != 2 || history.Observations[0] != rows[0] {
		t.Fatal("history", err)
	}
	current, err := s.CurrentListing(ctx, "l")
	if err != nil || current.Observation == nil || *current.Observation != rows[1] {
		t.Fatal("current", err)
	}
	events, _, err := s.ListAlertEvents(ctx, "l")
	if err != nil || len(events) != 0 {
		t.Fatal("retroactive alerts", err)
	}
	// Preexisting second-row identity forces a DB error after the first insert.
	if err := s.InsertPriceObservation(ctx, "csv:rollback:2", rows[1]); err != nil {
		t.Fatal(err)
	}
	if err := s.ImportObservations(ctx, "l", "rollback", rows); err == nil {
		t.Fatal("expected duplicate failure")
	}
	all, err := s.ListPriceObservations(ctx, "l")
	if err != nil || len(all) != 3 {
		t.Fatal("partial import committed", len(all), err)
	}
	if err := s.ImportObservations(ctx, "l", "invalid", []domain.PriceObservation{rows[0], {}}); err == nil {
		t.Fatal("invalid row accepted")
	}
	all, err = s.ListPriceObservations(ctx, "l")
	if err != nil || len(all) != 3 {
		t.Fatal("invalid import wrote", err)
	}
}
