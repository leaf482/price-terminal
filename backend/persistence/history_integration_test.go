//go:build integration

package persistence_test

import (
	"context"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"testing"
	"time"
)

func TestHistoryIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	store := persistence.New(freshDatabase(t, ctx))
	if err := store.InsertProduct(ctx, domain.Product{ID: "p"}); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertRetailer(ctx, domain.Retailer{ID: "r"}); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertListing(ctx, domain.Listing{ID: "l", ProductID: "p", RetailerID: "r", URL: "https://example.com/l"}); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 9, 29, 0, 0, 0, 123456789, time.UTC)
	empty, err := store.PriceHistory(ctx, "l", nil, at)
	if err != nil || len(empty.Observations) != 0 {
		t.Fatal(err)
	}
	for i := 0; i < 1002; i++ {
		o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: at.Add(time.Duration(i) * time.Nanosecond), Source: fmt.Sprintf("%04d", i), Stock: domain.StockUnknown})
		if err != nil {
			t.Fatal(err)
		}
		if err := store.InsertPriceObservation(ctx, fmt.Sprintf("%04d", i), o); err != nil {
			t.Fatal(err)
		}
	}
	end := at.Add(1001 * time.Nanosecond)
	all, err := store.PriceHistory(ctx, "l", nil, end)
	if err != nil || !all.Truncated || len(all.Observations) != 1000 {
		t.Fatal("bounded ALL", err)
	}
	if all.Observations[0].Source() != "0002" || all.Observations[999].Source() != "1001" {
		t.Fatal("order/window")
	}
	single, err := store.PriceHistory(ctx, "l", &at, at)
	if err != nil || len(single.Observations) != 1 || !single.Observations[0].ObservedAt().Equal(at) {
		t.Fatal("inclusive precision", err)
	}
	if _, ok := single.Observations[0].Currency(); ok {
		t.Fatal("stock-only currency")
	}
	money := domain.Money{MinorUnits: 0, Currency: domain.JPY}
	o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: at, Source: "tie", Stock: domain.StockInStock, OfferPrice: &money})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.InsertPriceObservation(ctx, "zz", o); err != nil {
		t.Fatal(err)
	}
	ties, err := store.PriceHistory(ctx, "l", &at, at)
	if err != nil || len(ties.Observations) != 2 || ties.Observations[1].Source() != "tie" {
		t.Fatal("ties", err)
	}
	m, ok := ties.Observations[1].OfferPrice()
	if !ok || m != money {
		t.Fatal("zero lost")
	}
}
