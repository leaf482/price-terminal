//go:build integration

package persistence_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"github.com/leaf482/price-terminal/backend/persistence"
	"github.com/leaf482/price-terminal/backend/provider"
	"reflect"
	"testing"
	"time"
)

func TestAlertIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := persistence.New(db)
	for _, err := range []error{s.InsertProduct(ctx, domain.Product{ID: "p"}), s.InsertRetailer(ctx, domain.Retailer{ID: "r"})} {
		if err != nil {
			t.Fatal(err)
		}
	}
	l := domain.Listing{ID: "l", ProductID: "p", RetailerID: "r", URL: "https://example.com/item"}
	if err := s.InsertListing(ctx, l); err != nil {
		t.Fatal(err)
	}
	threshold, drop := int64(100), int64(1000)
	alerts := []domain.PriceAlert{
		{ID: "target", ListingID: "l", Kind: "target", Currency: domain.USD, Threshold: &threshold, Enabled: true, RequireInStock: true},
		{ID: "drop", ListingID: "l", Kind: "drop", Currency: domain.USD, DropBasisPoints: &drop, Enabled: true, RequireInStock: true},
		{ID: "low", ListingID: "l", Kind: "historical_low", Currency: domain.USD, Enabled: true, RequireInStock: true},
	}
	for _, a := range alerts {
		if err := s.InsertAlert(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.ListAlerts(ctx, "l")
	if err != nil || len(got) != 3 {
		t.Fatal(got, err)
	}
	if !reflect.DeepEqual(got, []domain.PriceAlert{alerts[1], alerts[2], alerts[0]}) {
		t.Fatal("round trip", got)
	}
	invalid := alerts[0]
	invalid.ID = "invalid"
	invalid.Threshold = nil
	if s.InsertAlert(ctx, invalid) == nil {
		t.Fatal("invalid alert persisted")
	}
	invalid = alerts[0]
	invalid.ID = "missing"
	invalid.ListingID = "absent"
	if !errors.Is(s.InsertAlert(ctx, invalid), sql.ErrNoRows) {
		t.Fatal("missing listing accepted")
	}
	insert := func(id string, second int, price *domain.Money) {
		t.Helper()
		o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: time.Unix(int64(second), 123), Source: "fixture", Stock: domain.StockInStock, OfferPrice: price})
		if err != nil {
			t.Fatal(err)
		}
		fake := provider.NewFake(map[string]provider.FakeResponse{"l": {Observation: o}})
		if _, err = ingestion.New(fake, s).Ingest(ctx, id, l); err != nil {
			t.Fatal(err)
		}
	}
	events := func(want int) []persistence.AlertEvent {
		t.Helper()
		es, more, err := s.ListAlertEvents(ctx, "l")
		if err != nil || more || len(es) != want {
			t.Fatalf("events=%+v more=%v err=%v want=%d", es, more, err, want)
		}
		return es
	}
	insert("first", 1, &domain.Money{MinorUnits: 100, Currency: domain.USD})
	es := events(1)
	if es[0].Kind != "target" {
		t.Fatal("first triggered comparison alert")
	}
	insert("jpy", 2, &domain.Money{MinorUnits: 1, Currency: domain.JPY})
	events(1)
	insert("stock", 3, nil)
	events(1)
	insert("second", 4, &domain.Money{MinorUnits: 90, Currency: domain.USD})
	es = events(4)
	for _, e := range es {
		if e.ObservationID == "second" && e.Kind != "target" && (e.ComparisonMinorUnits == nil || *e.ComparisonMinorUnits != 100) {
			t.Fatal("wrong preceding comparison", e)
		}
	}
	if err := s.EvaluateAlerts(ctx, "second"); err != nil {
		t.Fatal(err)
	}
	events(4)
	insert("same later", 5, &domain.Money{MinorUnits: 90, Currency: domain.USD})
	events(5)
	a, err := s.SetAlertEnabled(ctx, "l", "target", false)
	if err != nil || a.Enabled {
		t.Fatal(a, err)
	}
	insert("disabled", 6, &domain.Money{MinorUnits: 90, Currency: domain.USD})
	events(5)
	insert("increase", 7, &domain.Money{MinorUnits: 110, Currency: domain.USD})
	events(5)
	// Equal timestamps use result ID ordering. Drop compares with 110, while
	// historical low compares with 90; these must not share one prior rule.
	insert("z-tie", 7, &domain.Money{MinorUnits: 95, Currency: domain.USD})
	es = events(6)
	if es[0].Kind != "drop" || *es[0].ComparisonMinorUnits != 110 {
		t.Fatal("wrong immediate predecessor", es[0])
	}
	insert("zero", 8, &domain.Money{MinorUnits: 0, Currency: domain.USD})
	es = events(8)
	for _, e := range es {
		if e.ObservationID == "zero" && (e.MinorUnits != 0 || e.ObservedAt != time.Unix(8, 123).UTC() || e.PriceBasis != "offer_price") {
			t.Fatal("event lost facts", e)
		}
	}
	if _, err := s.SetAlertEnabled(ctx, "other", "target", true); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("cross-listing toggle", err)
	}
	// A genuine PostgreSQL evaluation error happens after the observation commit.
	if _, err := s.SetAlertEnabled(ctx, "l", "target", true); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE price_alert_events RENAME TO test_hidden_events`); err != nil {
		t.Fatal(err)
	}
	insert("evaluation-failure", 9, &domain.Money{MinorUnits: 0, Currency: domain.USD})
	var retained int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM price_observations WHERE result_id='evaluation-failure'`).Scan(&retained); err != nil || retained != 1 {
		t.Fatal(retained, err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE test_hidden_events RENAME TO price_alert_events`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetAlertEnabled(ctx, "l", "target", false); err != nil {
		t.Fatal(err)
	}
	events(8)
	// Bounds include disabled configurations; an API read never hides active
	// configurations behind an arbitrary truncated subset.
	for i := 3; i < persistence.MaxAlerts; i++ {
		a := alerts[0]
		a.ID = fmt.Sprintf("extra-%03d", i)
		a.Enabled = false
		if err := s.InsertAlert(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	a = alerts[0]
	a.ID = "over-limit"
	if !errors.Is(s.InsertAlert(ctx, a), persistence.ErrAlertLimit) {
		t.Fatal("alert limit not enforced")
	}
	if got, err := s.ListAlerts(ctx, "l"); err != nil || len(got) != persistence.MaxAlerts {
		t.Fatal(len(got), err)
	}
	if _, err := s.SetAlertEnabled(ctx, "l", "target", true); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 101; i++ {
		insert(fmt.Sprintf("bounded-%03d", i), 20+i, &domain.Money{MinorUnits: 0, Currency: domain.USD})
	}
	if got, more, err := s.ListAlertEvents(ctx, "l"); err != nil || !more || len(got) != 100 {
		t.Fatal(len(got), more, err)
	}
}
