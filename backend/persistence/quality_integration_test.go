//go:build integration

package persistence_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"reflect"
	"testing"
	"time"
)

// Shared by data-quality and backup verification; all rows belong to a newly
// created disposable database, never the configured development database.
func seedReliability(t *testing.T, ctx context.Context, db *sql.DB) *persistence.Store {
	t.Helper()
	s := persistence.New(db)
	for _, err := range []error{s.InsertProduct(ctx, domain.Product{ID: "p", Name: "Test product", Brand: "Brand", Model: "Model"}), s.InsertRetailer(ctx, domain.Retailer{ID: "r", Name: "Test shop"}), s.InsertListing(ctx, domain.Listing{ID: "l", ProductID: "p", RetailerID: "r", URL: "https://example.com/item", RetailerProductID: "SKU"})} {
		if err != nil {
			t.Fatal(err)
		}
	}
	threshold := int64(100)
	a := domain.PriceAlert{ID: "target", ListingID: "l", Kind: "target", Currency: domain.USD, Threshold: &threshold, Enabled: true}
	if err := s.InsertAlert(ctx, a); err != nil {
		t.Fatal(err)
	}
	for i, id := range []string{"old", "bad"} {
		o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: time.Unix(int64(i+1), 123), Source: "fixture", Stock: domain.StockInStock, OfferPrice: &domain.Money{MinorUnits: int64(100 - i*90), Currency: domain.USD}})
		if err != nil {
			t.Fatal(err)
		}
		if err = s.InsertPriceObservation(ctx, id, o); err != nil {
			t.Fatal(err)
		}
		if err = s.EvaluateAlerts(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	p := domain.Promotion{ID: "promo", ListingID: "l", Source: "fixture", ObservedAt: time.Unix(1, 0), Kind: "fixed", Amount: &domain.Money{MinorUnits: 1, Currency: domain.USD}, Requirement: "unknown", Stacking: "unknown", Terms: "original evidence"}
	if err := s.InsertPromotion(ctx, p); err != nil {
		t.Fatal(err)
	}
	return s
}
func TestObservationInvalidationIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := seedReliability(t, ctx, db)
	before, err := s.GetObservationAudit(ctx, "bad")
	if err != nil {
		t.Fatal(err)
	}
	invalid, err := s.InvalidateObservation(ctx, "bad", "incorrect source price")
	if err != nil || invalid.InvalidatedAt == nil || !reflect.DeepEqual(invalid.Observation, before.Observation) {
		t.Fatal(invalid, err)
	}
	again, err := s.InvalidateObservation(ctx, "bad", "different retry reason")
	if err != nil || !reflect.DeepEqual(again, invalid) {
		t.Fatal("invalidation not first-write-wins", again, err)
	}
	for _, tt := range []struct {
		id, reason string
		want       error
	}{{"", "reason", persistence.ErrInvalidInvalidation}, {"bad", " ", persistence.ErrInvalidInvalidation}, {"missing", "reason", sql.ErrNoRows}} {
		if _, err := s.InvalidateObservation(ctx, tt.id, tt.reason); !errors.Is(err, tt.want) {
			t.Fatal(err)
		}
	}
	assertCurrent := func(want int64) {
		t.Helper()
		v, err := s.CurrentListing(ctx, "l")
		if err != nil || v.Observation == nil {
			t.Fatal(v, err)
		}
		m, _ := v.Observation.OfferPrice()
		if m.MinorUnits != want {
			t.Fatal(m)
		}
		p, err := s.CurrentProduct(ctx, "p")
		if err != nil || len(p) != 1 || !reflect.DeepEqual(p[0].Observation, v.Observation) {
			t.Fatal(p, err)
		}
	}
	assertCurrent(100)
	h, err := s.PriceHistory(ctx, "l", nil, time.Now())
	if err != nil || len(h.Observations) != 1 || h.Observations[0].ObservedAt() != time.Unix(1, 123).UTC() {
		t.Fatal(h, err)
	}
	audit, more, err := s.ListObservationAudit(ctx, "l")
	if err != nil || more || len(audit) != 2 || audit[0].InvalidatedAt == nil {
		t.Fatal(audit, err)
	}
	// A new alert must not evaluate the invalid record, and earlier invalid data
	// must not suppress a genuine later historical low or percentage drop.
	drop := int64(1000)
	for _, a := range []domain.PriceAlert{{ID: "late-target", ListingID: "l", Kind: "target", Currency: domain.USD, Threshold: func() *int64 { v := int64(100); return &v }(), Enabled: true}, {ID: "low", ListingID: "l", Kind: "historical_low", Currency: domain.USD, Enabled: true}, {ID: "drop", ListingID: "l", Kind: "drop", Currency: domain.USD, DropBasisPoints: &drop, Enabled: true}} {
		if err := s.InsertAlert(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.EvaluateAlerts(ctx, "bad"); err != nil {
		t.Fatal(err)
	}
	es, _, err := s.ListAlertEvents(ctx, "l")
	if err != nil || len(es) != 2 {
		t.Fatal("old events rewritten or invalid evaluated", es, err)
	}
	o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: time.Unix(3, 123), Source: "later", Stock: domain.StockInStock, OfferPrice: &domain.Money{MinorUnits: 80, Currency: domain.USD}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.InsertPriceObservation(ctx, "later", o); err != nil {
		t.Fatal(err)
	}
	if err = s.EvaluateAlerts(ctx, "later"); err != nil {
		t.Fatal(err)
	}
	assertCurrent(80)
	es, _, err = s.ListAlertEvents(ctx, "l")
	if err != nil || len(es) != 6 {
		t.Fatal(es, err)
	}
	if p, err := s.GetPromotion(ctx, "l", "promo"); err != nil || p.Terms != "original evidence" {
		t.Fatal(p, err)
	}
	for _, id := range []string{"old", "later"} {
		if _, err = s.InvalidateObservation(ctx, id, "retire from queries"); err != nil {
			t.Fatal(err)
		}
	}
	v, err := s.CurrentListing(ctx, "l")
	if err != nil || v.Observation != nil {
		t.Fatal(v, err)
	}
	h, err = s.PriceHistory(ctx, "l", nil, time.Now())
	if err != nil || len(h.Observations) != 0 {
		t.Fatal(h, err)
	}
	raw, err := s.ListPriceObservations(ctx, "l")
	if err != nil || len(raw) != 3 {
		t.Fatal("raw audit facts lost", raw, err)
	}
}
