//go:build integration

package persistence_test

import (
	"context"
	"github.com/leaf482/price-terminal/backend/persistence"
	"testing"
	"time"
)

func TestHomeCountsAndFailures(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := persistence.New(db)
	now := time.Now().UTC()
	counts, err := s.HomeCounts(ctx, now)
	if err != nil || counts != (persistence.HomeCounts{}) {
		t.Fatal(counts, err)
	}
	rows, err := s.RecentCollectionFailures(ctx)
	if err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
	s = seedReliability(t, ctx, db)
	if _, err := db.ExecContext(ctx, `UPDATE price_alert_events SET triggered_at=$1`, now); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO products(id,archived) VALUES ('constructor',true);
 INSERT INTO listings(id,product_id,retailer_id,url,tracking_enabled) VALUES ('__proto__','constructor','r','https://example.com/disabled',false)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO collection_attempts(id,listing_id,trigger,started_at,finished_at,outcome,error_summary)
 SELECT 'f-'||n,'l','manual',$1,$1,'provider_error','provider_error' FROM generate_series(1,7)n`, now)
	if err != nil {
		t.Fatal(err)
	}
	counts, err = s.HomeCounts(ctx, now)
	if err != nil || counts != (persistence.HomeCounts{ActiveProducts: 1, ArchivedProducts: 1, Listings: 2, TrackingEnabled: 1, TrackingDisabled: 1, CollectionErrors: 1, EnabledAlerts: 1, RecentlyTriggeredAlerts: 1}) {
		t.Fatal(counts, err)
	}
	rows, err = s.RecentCollectionFailures(ctx)
	if err != nil || len(rows) != 5 || rows[0].ID != "f-7" || rows[4].ID != "f-3" {
		t.Fatal(rows, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE listings SET tracking_enabled=false WHERE id='l'; UPDATE price_alerts SET enabled=false`); err != nil {
		t.Fatal(err)
	}
	counts, err = s.HomeCounts(ctx, now)
	if err != nil || counts.CollectionErrors != 0 || counts.EnabledAlerts != 0 || counts.TrackingDisabled != 2 || counts.RecentlyTriggeredAlerts != 1 {
		t.Fatal(counts, err)
	}
}
