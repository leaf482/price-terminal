//go:build integration

package persistence_test

import (
	"context"
	"github.com/leaf482/price-terminal/backend/persistence"
	"testing"
	"time"
)

func TestRecentPriceChanges(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := persistence.New(db)
	empty, err := s.RecentPriceChanges(ctx)
	if err != nil || len(empty.Changes) != 0 {
		t.Fatal(empty, err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO products(id,name) VALUES ('constructor','Camera'); INSERT INTO retailers(id,name) VALUES ('__proto__','Shop'); INSERT INTO listings(id,product_id,retailer_id,url) VALUES ('toString','constructor','__proto__','https://example.com/item');
 INSERT INTO price_observations(result_id,listing_id,observed_at,observed_at_ns_remainder,source,stock,currency,offer_price,sale_price,msrp_source) VALUES
 ('a','toString','2026-01-01',1,'manual','in_stock','USD',100,999,''),
 ('b','toString','2026-01-02',1,'manual','unknown',NULL,NULL,NULL,''),
 ('c','toString','2026-01-03',1,'manual','in_stock','JPY',800,NULL,''),
 ('d','toString','2026-01-04',1,'manual','in_stock','USD',1,NULL,''),
 ('e','toString','2026-01-05',1,'manual','in_stock','USD',NULL,75,''),
 ('f','toString','2026-01-06',1,'manual','in_stock','USD',75,NULL,''),
 ('g','toString','2026-01-07',1,'manual','in_stock','USD',0,NULL,''),
 ('h','toString','2026-01-08',1,'manual','in_stock','USD',100,NULL,''),
 ('i','toString','2026-01-08',2,'manual','in_stock','JPY',900,NULL,''),
 ('j','toString','2026-01-08',2,'manual','in_stock','JPY',900,NULL,'')`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.InvalidateObservation(ctx, "d", "bad price"); err != nil {
		t.Fatal(err)
	}
	got, err := s.RecentPriceChanges(ctx)
	if err != nil || len(got.Changes) != 6 {
		t.Fatal(got, err)
	}
	for i, id := range []string{"j", "i", "h", "g", "f", "e"} {
		if got.Changes[i].Current.ID != id {
			t.Fatal(got)
		}
	}
	last := got.Changes[5]
	if last.Previous.ID != "a" || last.ChangeMinor != "-25" || *last.Percentage != "-25.00" || last.ProductID != "constructor" || last.RetailerID != "__proto__" {
		t.Fatal(last)
	}
	if got.Changes[1].Previous.ID != "c" || got.Changes[1].Currency != "JPY" || got.Changes[2].Percentage != nil || got.Changes[3].Current.MinorUnits != "0" || got.Changes[4].Direction != "unchanged" || got.Changes[0].Current.ObservedAt.Nanosecond() != 2 {
		t.Fatal(got)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO price_observations(result_id,listing_id,observed_at,observed_at_ns_remainder,source,stock,currency,offer_price,msrp_source) SELECT 'z-'||lpad(n::text,3,'0'),'toString','2026-02-01'::timestamptz,n,'manual','unknown','USD',100,'' FROM generate_series(1,101)n`)
	if err != nil {
		t.Fatal(err)
	}
	got, err = s.RecentPriceChanges(ctx)
	if err != nil || !got.Truncated || len(got.Changes) != 100 || got.Changes[0].Current.ID != "z-101" || got.Changes[99].Current.ID != "z-002" || got.Changes[99].Previous.ID != "z-001" {
		t.Fatal(got, err)
	}
}
