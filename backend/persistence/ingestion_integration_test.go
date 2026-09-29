//go:build integration

package persistence_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"github.com/leaf482/price-terminal/backend/persistence"
	"github.com/leaf482/price-terminal/backend/provider"
)

func TestSingleListingIngestionIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	store := persistence.New(db)
	if err := store.InsertProduct(ctx, domain.Product{ID: "p"}); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertRetailer(ctx, domain.Retailer{ID: "r"}); err != nil {
		t.Fatal(err)
	}
	listing := domain.Listing{ID: "l", ProductID: "p", RetailerID: "r", URL: "https://example.com/item", RetailerProductID: "sku"}
	if err := store.InsertListing(ctx, listing); err != nil {
		t.Fatal(err)
	}
	input := domain.PriceObservationInput{ListingID: "l", ObservedAt: time.Date(2026, 9, 28, 12, 0, 0, 123, time.UTC), Source: "fixture evidence", Stock: domain.StockInStock, OfferPrice: &domain.Money{MinorUnits: 100, Currency: domain.USD}}
	makeObservation := func() domain.PriceObservation {
		o, err := domain.NewPriceObservation(input)
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	first := makeObservation()
	fake := provider.NewFake(map[string]provider.FakeResponse{"l": {Observation: first}})
	i := ingestion.New(fake, store)
	collected, err := i.Ingest(ctx, "first", listing)
	if err != nil {
		t.Fatal(err)
	}
	assertDuplicate := func(err error) {
		var pgError *pgconn.PgError
		if !errors.As(err, &pgError) || pgError.Code != "23505" {
			t.Fatalf("want preserved uniqueness error, got %v", err)
		}
	}
	assertDuplicate(i.Persist(ctx, collected))
	input.ObservedAt = input.ObservedAt.Add(time.Minute)
	later := makeObservation()
	next := ingestion.New(provider.NewFake(map[string]provider.FakeResponse{"l": {Observation: later}}), store)
	// Reusing an ID for different facts also fails without overwriting history.
	_, err = next.Ingest(ctx, "first", listing)
	assertDuplicate(err)
	if _, err := next.Ingest(ctx, "later", listing); err != nil {
		t.Fatal(err)
	}
	want := []domain.PriceObservation{first, later}
	for _, mode := range []string{"stock-only", "zero"} {
		input.ObservedAt = input.ObservedAt.Add(time.Minute)
		input.OfferPrice = nil
		if mode == "zero" {
			input.OfferPrice = &domain.Money{Currency: domain.JPY}
		}
		o := makeObservation()
		if _, err := ingestion.New(provider.NewFake(map[string]provider.FakeResponse{"l": {Observation: o}}), store).Ingest(ctx, mode, listing); err != nil {
			t.Fatal(err)
		}
		want = append(want, o)
	}
	// Every failure below must leave both catalog and observation counts unchanged.
	counts := func() [4]int {
		var n [4]int
		if err := db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM products), (SELECT count(*) FROM retailers),
			(SELECT count(*) FROM listings), (SELECT count(*) FROM price_observations)`).Scan(&n[0], &n[1], &n[2], &n[3]); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := counts()
	cause := errors.New("collection failed")
	failed := ingestion.New(provider.NewFake(map[string]provider.FakeResponse{"l": {Err: cause}}), store)
	if _, err := failed.Ingest(ctx, "failure", listing); !errors.Is(err, cause) {
		t.Fatalf("failure cause=%v", err)
	}
	missing := listing
	missing.ID = "missing"
	input.ListingID = missing.ID
	missingObservation := makeObservation()
	if _, err := ingestion.New(provider.NewFake(map[string]provider.FakeResponse{"missing": {Observation: missingObservation}}), store).Ingest(ctx, "missing", missing); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing listing=%v", err)
	}
	for _, change := range []func(*domain.Listing){
		func(l *domain.Listing) { l.ProductID = "missing" }, func(l *domain.Listing) { l.RetailerID = "missing" },
	} {
		invalid := listing
		change(&invalid)
		if _, err := i.Ingest(ctx, "invalid-relationship", invalid); err == nil {
			t.Fatal("invalid relationship accepted")
		}
	}
	if after := counts(); after != before {
		t.Fatalf("partial writes: %v -> %v", before, after)
	}
	got, err := store.ListPriceObservations(ctx, "l")
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("history=%+v err=%v want=%+v", got, err, want)
	}
}
