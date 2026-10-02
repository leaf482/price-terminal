//go:build integration

package persistence_test

import (
	"context"
	"testing"
	"time"
)

func TestAuditHistoryBoundsAndOrdering(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := seedReliability(t, ctx, db)
	_, err := db.ExecContext(ctx, `INSERT INTO price_observations(result_id,listing_id,observed_at,observed_at_ns_remainder,source,stock,msrp_source) SELECT 'tie-'||lpad(n::text,3,'0'),'l','2026-01-01'::timestamptz,789,'manual: CSV import evidence','unknown','' FROM generate_series(1,101)n`)
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.GetObservationAudit(ctx, "tie-101")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.InvalidateObservation(ctx, "tie-101", "incorrect source"); err != nil {
		t.Fatal(err)
	}
	rows, more, err := s.ListObservationAudit(ctx, "l")
	if err != nil || !more || len(rows) != 100 {
		t.Fatal(len(rows), more, err)
	}
	if rows[0].ID != "tie-101" || rows[1].ID != "tie-100" || rows[99].ID != "tie-002" || rows[0].InvalidatedAt == nil || rows[0].Reason != "incorrect source" || rows[0].Observation != before.Observation {
		t.Fatal("audit ordering/facts", rows[0])
	}
	current, err := s.CurrentListing(ctx, "l")
	if err != nil || current.Observation == nil {
		t.Fatal(current, err)
	}
	history, err := s.PriceHistory(ctx, "l", nil, time.Now())
	if err != nil || len(history.Observations) != 102 {
		t.Fatal("normal history must exclude invalid row", len(history.Observations), err)
	}
	exported, err := s.ExportObservationAudit(ctx, "l")
	if err != nil || len(exported) != 103 {
		t.Fatal("export must retain all rows", len(exported), err)
	}
}
