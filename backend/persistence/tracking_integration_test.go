//go:build integration

package persistence_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"github.com/leaf482/price-terminal/backend/persistence"
	"os/exec"
	"testing"
	"time"
)

func TestTrackingPersistenceIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := persistence.New(db)
	if err := s.InsertProduct(ctx, domain.Product{ID: "p"}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertRetailer(ctx, domain.Retailer{ID: "r"}); err != nil {
		t.Fatal(err)
	}
	// Exercise migration of an existing Listing, not just new insert defaults.
	var name string
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	migrate := func(actions ...string) {
		t.Helper()
		out, err := exec.CommandContext(ctx, "goose", append([]string{"-env", "none", "-dir", "../migrations", "postgres", "dbname=" + name}, actions...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v %s", actions, err, out)
		}
	}
	migrate("down-to", "6")
	if _, err := db.ExecContext(ctx, `INSERT INTO listings(id,product_id,retailer_id,url,retailer_product_id) VALUES('l','p','r','https://example.com/l','sku')`); err != nil {
		t.Fatal(err)
	}
	migrate("up")
	listing, err := s.GetListing(ctx, "l")
	if err != nil || !listing.TrackingEnabled() {
		t.Fatal("legacy default", listing, err)
	}
	threshold := int64(100)
	if err := s.InsertAlert(ctx, domain.PriceAlert{ID: "a", ListingID: "l", Kind: "target", Currency: domain.USD, Threshold: &threshold, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC().Add(-time.Minute)
	money := &domain.Money{MinorUnits: 0, Currency: domain.USD}
	o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: at, Source: "fixture", Stock: domain.StockInStock, OfferPrice: money})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.InsertCollectedObservation(ctx, "first", o); err != nil {
		t.Fatal(err)
	}
	if err := s.EvaluateAlerts(ctx, "first"); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertPromotion(ctx, domain.Promotion{ID: "promotion", ListingID: "l", ObservedAt: at, Source: "fixture", Terms: "explicit zero discount", Kind: "fixed", Amount: money, Requirement: "none", Stacking: "allowed"}); err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{false, false, true, false} {
		if err := s.SetListingTracking(ctx, "l", enabled); err != nil {
			t.Fatal(err)
		}
		got, err := s.GetListing(ctx, "l")
		if err != nil || got.TrackingEnabled() != enabled {
			t.Fatal(got, err)
		}
		got.TrackingDisabled = false
		if got != listing {
			t.Fatal("metadata changed")
		}
	}
	if err := s.InsertCollectedObservation(ctx, "blocked", o); !errors.Is(err, domain.ErrTrackingDisabled) {
		t.Fatal("disabled collection accepted", err)
	}
	disabled, err := s.GetListing(ctx, "l")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ingestion.New(nil, s).Record(ctx, "manual", disabled, o); err != nil {
		t.Fatal("explicit manual blocked", err)
	}
	history, err := s.ListPriceObservations(ctx, "l")
	if err != nil || len(history) != 2 {
		t.Fatal(history, err)
	}
	current, err := s.CurrentListing(ctx, "l")
	if err != nil || current.Observation == nil || !current.Listing.TrackingDisabled {
		t.Fatal("disabled current missing", err)
	}
	if _, err := s.GetPromotion(ctx, "l", "promotion"); err != nil {
		t.Fatal("promotion lost", err)
	}
	alerts, err := s.ListAlerts(ctx, "l")
	if err != nil || len(alerts) != 1 {
		t.Fatal("alerts lost", err)
	}
	events, _, err := s.ListAlertEvents(ctx, "l")
	if err != nil || len(events) != 2 {
		t.Fatal("events lost", err)
	}
	if err := s.SetListingTracking(ctx, "l", true); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertCollectedObservation(ctx, "resumed", o); err != nil {
		t.Fatal("re-enable blocked", err)
	}
	if err := s.SetListingTracking(ctx, "missing", true); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
}
