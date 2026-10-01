//go:build integration

package persistence_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

func TestCSVExportIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := seedReliability(t, ctx, db)
	if _, err := s.InvalidateObservation(ctx, "bad", "wrong, \"price\"\naudit"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"tie-z", "tie-a"} {
		o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: time.Unix(2, 124), Source: "fixture", Stock: domain.StockUnknown})
		if err != nil {
			t.Fatal(err)
		}
		if err = s.InsertPriceObservation(ctx, id, o); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := s.ExportObservationAudit(ctx, "l")
	if err != nil || len(rows) != 4 {
		t.Fatal(rows, err)
	}
	for i, id := range []string{"old", "bad", "tie-a", "tie-z"} {
		if rows[i].ID != id {
			t.Fatal(rows)
		}
	}
	if rows[1].InvalidatedAt == nil || rows[1].Reason != "wrong, \"price\"\naudit" || rows[2].Observation.ObservedAt().Nanosecond() != 124 {
		t.Fatal(rows)
	}
	if _, present := rows[2].Observation.Currency(); present {
		t.Fatal("invented stock-only currency")
	}
	empty, err := s.ExportObservationAudit(ctx, "absent")
	if err != nil || len(empty) != 0 {
		t.Fatal(empty, err)
	}
	// Fill exactly to the bound, then exceed it. Only this disposable DB is used.
	_, err = db.ExecContext(ctx, `INSERT INTO price_observations(result_id,listing_id,observed_at,observed_at_ns_remainder,source,stock,msrp_source) SELECT 'bulk-'||n,'l','2026-01-01'::timestamptz,0,'fixture','unknown','' FROM generate_series(1,$1) n`, persistence.MaxExportRows-4)
	if err != nil {
		t.Fatal(err)
	}
	rows, err = s.ExportObservationAudit(ctx, "l")
	if err != nil || len(rows) != persistence.MaxExportRows {
		t.Fatal(len(rows), err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO price_observations(result_id,listing_id,observed_at,observed_at_ns_remainder,source,stock,msrp_source) VALUES ('overflow','l',now(),0,'fixture','unknown','')`)
	if err != nil {
		t.Fatal(err)
	}
	rows, err = s.ExportObservationAudit(ctx, "l")
	if !errors.Is(err, persistence.ErrExportLimit) || rows != nil {
		t.Fatal("truncated export", len(rows), err)
	}
}
