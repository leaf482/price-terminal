//go:build integration

package persistence_test

import (
	"context"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
	"testing"
	"time"
)

func TestCollectionAttemptHistory(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := seedReliability(t, ctx, db)
	for i := 0; i < 55; i++ {
		a := domain.CollectionAttempt{ID: fmt.Sprintf("attempt-%02d", i), ListingID: "l", Trigger: "scheduled", StartedAt: time.Unix(1, 0), FinishedAt: time.Unix(2, 0), Outcome: "provider_error", ErrorSummary: "provider_error"}
		if err := s.InsertCollectionAttempt(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := s.ListCollectionAttempts(ctx, "l")
	if err != nil || len(rows) != 50 || rows[0].ID != "attempt-54" || rows[49].ID != "attempt-05" {
		t.Fatal(rows, err)
	}
	a := domain.CollectionAttempt{ID: "success", ListingID: "l", Trigger: "manual", StartedAt: time.Unix(3, 0), FinishedAt: time.Unix(4, 0), Outcome: "success", ObservationID: "old"}
	if err := s.InsertCollectionAttempt(ctx, a); err != nil {
		t.Fatal(err)
	}
	rows, err = s.ListCollectionAttempts(ctx, "l")
	if err != nil || rows[0].ObservationID != "old" {
		t.Fatal(rows, err)
	}
	a.ID = "bad-ref"
	a.ObservationID = "missing"
	if s.InsertCollectionAttempt(ctx, a) == nil {
		t.Fatal("missing observation accepted")
	}
	rows, err = s.ListCollectionAttempts(ctx, "missing")
	if err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
}
