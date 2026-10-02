//go:build integration

package persistence_test

import (
	"context"
	"github.com/leaf482/price-terminal/backend/domain"
	"testing"
	"time"
)

func TestCollectionHealthIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := seedReliability(t, ctx, db)
	if _, err := s.InvalidateObservation(ctx, "bad", "bad facts"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"constructor", "__proto__", "toString"} {
		if err := s.InsertListing(ctx, domain.Listing{ID: id, ProductID: "p", RetailerID: "r", URL: "https://example.com/" + id, TrackingDisabled: id == "toString"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, a := range []domain.CollectionAttempt{
		{ID: "success", ListingID: "l", Trigger: "manual", StartedAt: time.Unix(4, 0), FinishedAt: time.Unix(5, 0), Outcome: "success", ObservationID: "old"},
		{ID: "failure-a", ListingID: "l", Trigger: "manual", StartedAt: time.Unix(6, 0), FinishedAt: time.Unix(7, 0), Outcome: "provider_error", ErrorSummary: "provider_error"},
		{ID: "failure-z", ListingID: "l", Trigger: "scheduled", StartedAt: time.Unix(6, 0), FinishedAt: time.Unix(8, 0), Outcome: "cancelled", ErrorSummary: "cancelled"},
	} {
		if err := s.InsertCollectionAttempt(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.CollectionHealth(ctx)
	if err != nil || len(got.Listings) != 4 {
		t.Fatal(got, err)
	}
	v := got.Listings[2]
	if v.ListingID != "l" || v.Outcome != "cancelled" || v.ObservedAt.Unix() != 1 || v.SuccessfulAt.Unix() != 5 || v.AttemptedAt.Unix() != 6 || got.Counts.Disabled != 1 || got.Counts.Never != 2 {
		t.Fatal(got)
	}
	if got.Listings[0].ObservedAt != nil || got.Listings[0].AttemptedAt != nil {
		t.Fatal("fabricated timestamps")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO listings(id,product_id,retailer_id,url,retailer_product_id) SELECT 'z-'||lpad(n::text,3,'0'),'p','r','https://example.com/z/'||n,'' FROM generate_series(1,101)n`); err != nil {
		t.Fatal(err)
	}
	got, err = s.CollectionHealth(ctx)
	if err != nil || len(got.Listings) != 100 || !got.Truncated || got.Counts.Total != 100 || got.Listings[99].ListingID != "z-096" {
		t.Fatal(got, err)
	}
}
