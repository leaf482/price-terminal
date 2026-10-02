//go:build integration

package persistence_test

import (
	"context"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"testing"
	"time"
)

func TestAlertOverviewIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	empty, err := persistence.New(db).AlertOverview(ctx)
	if err != nil || len(empty.Alerts) != 0 || len(empty.Events) != 0 {
		t.Fatal(empty, err)
	}
	s := seedReliability(t, ctx, db)
	for _, err := range []error{s.InsertProduct(ctx, domain.Product{ID: "constructor", Name: "Camera"}), s.InsertRetailer(ctx, domain.Retailer{ID: "toString", Name: "Shop"}), s.InsertListing(ctx, domain.Listing{ID: "__proto__", ProductID: "constructor", RetailerID: "toString", URL: "https://example.com/new"})} {
		if err != nil {
			t.Fatal(err)
		}
	}
	zero, drop := int64(0), int64(1000)
	for _, a := range []domain.PriceAlert{{ID: "constructor", ListingID: "__proto__", Kind: "target", Currency: domain.USD, Threshold: &zero, Enabled: true}, {ID: "__proto__", ListingID: "__proto__", Kind: "drop", Currency: domain.USD, DropBasisPoints: &drop}, {ID: "toString", ListingID: "__proto__", Kind: "historical_low", Currency: domain.JPY, Enabled: true, RequireInStock: true}} {
		if err := s.InsertAlert(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	out, err := s.AlertOverview(ctx)
	if err != nil || len(out.Alerts) != 4 || len(out.Events) != 2 {
		t.Fatal(out, err)
	}
	for _, a := range out.Alerts {
		if a.Alert.ID == "target" {
			if a.LastTriggeredAt == nil {
				t.Fatal("lost event time")
			}
			continue
		}
		if a.ProductID != "constructor" || a.ProductName != "Camera" || a.RetailerName != "Shop" || a.LastTriggeredAt != nil {
			t.Fatal(a)
		}
		if a.Alert.ID == "constructor" && (a.Alert.Threshold == nil || *a.Alert.Threshold != 0) {
			t.Fatal(a)
		}
	}
	if out.Events[0].TriggeredAt.Before(out.Events[1].TriggeredAt) {
		t.Fatal("events out of order")
	}
	if _, err := s.SetAlertEnabled(ctx, "__proto__", "constructor", false); err != nil {
		t.Fatal(err)
	}
	out, err = s.AlertOverview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range out.Alerts {
		if a.Alert.ID == "constructor" && a.Alert.Enabled {
			t.Fatal("toggle not visible")
		}
	}
	_, err = db.ExecContext(ctx, `INSERT INTO price_alerts(id,listing_id,kind,currency,threshold_minor_units,enabled,require_in_stock) SELECT 'bound-'||lpad(n::text,3,'0'),'l','target','USD',100,true,false FROM generate_series(1,98)n`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO price_alert_events(alert_id,listing_id,observation_id,kind,triggered_at,minor_units,currency,price_basis) SELECT 'bound-'||lpad(n::text,3,'0'),'l','old','target','2099-01-01'::timestamptz,100,'USD','offer_price' FROM generate_series(1,23)n`)
	if err != nil {
		t.Fatal(err)
	}
	out, err = s.AlertOverview(ctx)
	if err != nil || len(out.Alerts) != 100 || !out.AlertsTruncated || len(out.Events) != 20 || !out.EventsTruncated {
		t.Fatal(out, err)
	}
	if out.Events[0].AlertID != "bound-001" || out.Events[19].AlertID != "bound-020" {
		t.Fatal("tie order", out.Events)
	}
	for i := 1; i < len(out.Alerts); i++ {
		if out.Alerts[i-1].Alert.ID >= out.Alerts[i].Alert.ID {
			t.Fatal("alert order")
		}
	}
}
