//go:build integration

package persistence_test

import (
	"context"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"github.com/leaf482/price-terminal/backend/persistence"
	"testing"
	"time"
)

func TestManualRecordIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s := persistence.New(freshDatabase(t, ctx))
	l := domain.Listing{ID: "l", ProductID: "p", RetailerID: "r", URL: "https://example.com/l"}
	for _, err := range []error{s.InsertProduct(ctx, domain.Product{ID: "p"}), s.InsertRetailer(ctx, domain.Retailer{ID: "r"}), s.InsertListing(ctx, l)} {
		if err != nil {
			t.Fatal(err)
		}
	}
	threshold := int64(100)
	if err := s.InsertAlert(ctx, domain.PriceAlert{ID: "a", ListingID: "l", Kind: "target", Currency: domain.USD, Threshold: &threshold, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: at, Source: "manual: receipt", Stock: domain.StockUnknown, OfferPrice: &domain.Money{MinorUnits: 0, Currency: domain.USD}})
	if err != nil {
		t.Fatal(err)
	}
	i := ingestion.New(nil, s)
	collected, err := i.Record(ctx, "manual-entry", l, o)
	if err != nil {
		t.Fatal(err)
	}
	if err := i.Persist(ctx, collected); err == nil {
		t.Fatal("duplicate accepted")
	}
	current, err := s.CurrentListing(ctx, "l")
	if err != nil || current.Observation == nil || *current.Observation != o {
		t.Fatal("current mismatch", err)
	}
	history, err := s.PriceHistory(ctx, "l", nil, at.Add(time.Hour))
	if err != nil || len(history.Observations) != 1 || history.Observations[0] != o {
		t.Fatal("history mismatch", err)
	}
	events, _, err := s.ListAlertEvents(ctx, "l")
	if err != nil || len(events) != 1 {
		t.Fatal("alert did not evaluate", err)
	}
	stock, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: at.Add(time.Minute), Source: "manual: stock check", Stock: domain.StockOutOfStock})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = i.Record(ctx, "stock-entry", l, stock); err != nil {
		t.Fatal(err)
	}
	current, err = s.CurrentListing(ctx, "l")
	if err != nil || current.Observation == nil || *current.Observation != stock {
		t.Fatal("stock-only current", err)
	}
	history, err = s.PriceHistory(ctx, "l", nil, at.Add(time.Hour))
	if err != nil || len(history.Observations) != 2 {
		t.Fatal("history lost", err)
	}
}
