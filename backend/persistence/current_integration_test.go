//go:build integration

package persistence_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"github.com/leaf482/price-terminal/backend/provider"
)

func TestCurrentPricesIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	store := persistence.New(freshDatabase(t, ctx))
	for _, p := range []string{"p", "empty", "large"} {
		if err := store.InsertProduct(ctx, domain.Product{ID: p}); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range []string{"r1", "r2"} {
		if err := store.InsertRetailer(ctx, domain.Retailer{ID: r}); err != nil {
			t.Fatal(err)
		}
	}
	for _, l := range []domain.Listing{
		{ID: "a", ProductID: "p", RetailerID: "r1", URL: "https://example.com/a", RetailerProductID: "sku"},
		{ID: "b", ProductID: "p", RetailerID: "r2", URL: "https://example.com/b"},
		{ID: "c", ProductID: "p", RetailerID: "r1", URL: "https://example.com/c"},
	} {
		if err := store.InsertListing(ctx, l); err != nil {
			t.Fatal(err)
		}
	}
	at := time.Date(2020, 1, 1, 0, 0, 0, 123456789, time.UTC)
	money := func(n int64, c domain.Currency) *domain.Money { return &domain.Money{MinorUnits: n, Currency: c} }
	insert := func(id string, input domain.PriceObservationInput) domain.PriceObservation {
		t.Helper()
		o, err := domain.NewPriceObservation(input)
		if err != nil {
			t.Fatal(err)
		}
		if err = store.InsertPriceObservation(ctx, id, o); err != nil {
			t.Fatal(err)
		}
		return o
	}
	full := domain.PriceObservationInput{ListingID: "a", ObservedAt: at, Source: "fixture", Stock: domain.StockInStock, MSRP: money(999, domain.USD), MSRPSource: "manufacturer", RetailerListPrice: money(800, domain.USD), SalePrice: money(500, domain.USD), OfferPrice: money(0, domain.USD)}
	want := insert("z", full)
	older := full
	older.ObservedAt = at.Add(-time.Nanosecond)
	older.Source = "older"
	insert("zz", older)
	tie := full
	tie.Source = "tie-loser"
	insert("A", tie)
	got, err := store.CurrentListing(ctx, "a")
	if err != nil || got.Observation == nil || *got.Observation != want || got.Listing.RetailerProductID != "sku" {
		t.Fatalf("latest coherent snapshot: %+v %v", got, err)
	}
	stock := domain.PriceObservationInput{ListingID: "a", ObservedAt: at.Add(time.Second), Source: "stock", Stock: domain.StockOutOfStock}
	want = insert("stock", stock)
	got, err = store.CurrentListing(ctx, "a")
	if err != nil || got.Observation == nil || *got.Observation != want {
		t.Fatalf("stock-only latest: %+v %v", got, err)
	}
	if _, ok := got.Observation.Currency(); ok {
		t.Fatal("invented currency")
	}
	insert("jpy", domain.PriceObservationInput{ListingID: "b", ObservedAt: at, Source: "jpy", Stock: domain.StockInStock, OfferPrice: money(0, domain.JPY)})
	values, err := store.CurrentProduct(ctx, "p")
	if err != nil || len(values) != 3 {
		t.Fatalf("product: %v %v", values, err)
	}
	if values[0].Listing.ID != "a" || values[1].Listing.ID != "b" || values[2].Listing.ID != "c" || values[2].Observation != nil {
		t.Fatal("ordering/missing history")
	}
	m, ok := values[1].Observation.OfferPrice()
	if !ok || m.MinorUnits != 0 || m.Currency != domain.JPY {
		t.Fatal("JPY zero")
	}
	if !values[1].Observation.ObservedAt().Equal(at) {
		t.Fatal("timestamp changed")
	}
	empty, err := store.CurrentProduct(ctx, "empty")
	if err != nil || len(empty) != 0 {
		t.Fatal("empty product", err)
	}
	if _, err := store.CurrentListing(ctx, "absent"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("missing listing", err)
	}
	for i := 0; i <= persistence.MaxCurrentListings; i++ {
		id := fmt.Sprintf("large-%03d", i)
		if err := store.InsertListing(ctx, domain.Listing{ID: id, ProductID: "large", RetailerID: "r1", URL: "https://example.com/" + id}); err != nil {
			t.Fatal(err)
		}
		if i == persistence.MaxCurrentListings-1 {
			v, e := store.CurrentProduct(ctx, "large")
			if e != nil || len(v) != 100 {
				t.Fatal("exact bound", e)
			}
		}
	}
	if v, e := store.CurrentProduct(ctx, "large"); !errors.Is(e, persistence.ErrTooManyListings) || v != nil {
		t.Fatal("must not truncate", e)
	}
	canceled, stop := context.WithCancel(ctx)
	stop()
	if _, e := store.CurrentProduct(canceled, "p"); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}

	t.Run("runtime persists healthy work despite failure", func(t *testing.T) {
		fake := provider.NewFake(map[string]provider.FakeResponse{
			"a": {Observation: want}, "b": {Err: errors.New("fixture failure")},
		})
		runtime, err := collector.New(store, []collector.Target{{ListingID: "b", Provider: fake}, {ListingID: "a", Provider: fake}}, time.Hour, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		beforeA, err := store.ListPriceObservations(ctx, "a")
		if err != nil {
			t.Fatal(err)
		}
		beforeB, err := store.ListPriceObservations(ctx, "b")
		if err != nil {
			t.Fatal(err)
		}
		runCtx, stop := context.WithTimeout(ctx, 5*time.Second)
		done := make(chan error, 1)
		go func() { done <- runtime.Run(runCtx) }()
		defer func() { stop(); <-done }()
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for runtime.Status("a").State != "success" {
			select {
			case <-runCtx.Done():
				t.Fatal("collector did not complete", runCtx.Err())
			case <-ticker.C:
			}
		}
		if runtime.Status("b").State != "failed" {
			t.Fatal("failure not isolated")
		}
		a, err := store.ListPriceObservations(ctx, "a")
		if err != nil {
			t.Fatal(err)
		}
		b, err := store.ListPriceObservations(ctx, "b")
		if err != nil {
			t.Fatal(err)
		}
		if len(a) != len(beforeA)+1 || len(b) != len(beforeB) {
			t.Fatal("unexpected observation writes")
		}
	})
}
