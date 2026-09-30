//go:build integration

package persistence_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"reflect"
	"testing"
	"time"
)

func TestPromotionIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	store := persistence.New(freshDatabase(t, ctx))
	if err := store.InsertProduct(ctx, domain.Product{ID: "p"}); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertRetailer(ctx, domain.Retailer{ID: "r"}); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertListing(ctx, domain.Listing{ID: "l", ProductID: "p", RetailerID: "r", URL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 29, 0, 0, 0, 123456789, time.UTC)
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	p := domain.Promotion{ID: "e1", ListingID: "l", Source: "official terms", ObservedAt: now, StartsAt: &start, EndsAt: &end, Kind: "fixed", Amount: &domain.Money{MinorUnits: 0, Currency: domain.JPY}, Requirement: "unknown", Stacking: "unknown", Terms: "CODE and eligibility unknown"}
	if err := store.InsertPromotion(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetPromotion(ctx, "l", "e1")
	if err != nil || !reflect.DeepEqual(got, p) {
		t.Fatal("round trip", got, err)
	}
	changed := p
	changed.Terms = "new evidence"
	var pg *pgconn.PgError
	if err := store.InsertPromotion(ctx, changed); !errors.As(err, &pg) || pg.Code != "23505" {
		t.Fatal("duplicate must fail", err)
	}
	changed.ID = "e2"
	changed.ObservedAt = now.Add(time.Nanosecond)
	changed.Kind = "percentage"
	changed.Amount = nil
	n := int64(1250)
	changed.BasisPoints = &n
	changed.StartsAt = nil
	changed.EndsAt = nil
	if err := store.InsertPromotion(ctx, changed); err != nil {
		t.Fatal(err)
	}
	original, err := store.GetPromotion(ctx, "l", "e1")
	if err != nil || !reflect.DeepEqual(original, p) {
		t.Fatal("history changed", err)
	}
	got, err = store.GetPromotion(ctx, "l", "e2")
	if err != nil || !reflect.DeepEqual(got, changed) {
		t.Fatal("optional/percentage roundtrip", got, err)
	}
	rows, more, err := store.RelevantPromotions(ctx, "l", now.Add(time.Second))
	if err != nil || more || len(rows) != 2 || rows[0].ID != "e2" {
		t.Fatal("list", rows, err)
	}
	rows, _, err = store.RelevantPromotions(ctx, "l", end)
	if err != nil || len(rows) != 1 || rows[0].ID != "e2" {
		t.Fatal("expiry", rows, err)
	}
	p.ID = "bad-fk"
	p.ListingID = "absent"
	if err := store.InsertPromotion(ctx, p); !errors.As(err, &pg) || pg.Code != "23503" {
		t.Fatal("FK", err)
	}
	p.ID = ""
	if err := store.InsertPromotion(ctx, p); err == nil {
		t.Fatal("invalid persisted")
	}
	for i := 0; i < persistence.MaxPromotions; i++ {
		copy := changed
		copy.ID = fmt.Sprintf("bounded-%03d", i)
		if err := store.InsertPromotion(ctx, copy); err != nil {
			t.Fatal(err)
		}
	}
	rows, more, err = store.RelevantPromotions(ctx, "l", end)
	if err != nil || !more || len(rows) != persistence.MaxPromotions {
		t.Fatal("bounded relevant query", len(rows), more, err)
	}
}
