package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

func TestObservationValidationBeforeDatabaseAccess(t *testing.T) {
	store := persistence.New(nil)
	ctx := context.Background()
	valid, err := domain.NewPriceObservation(domain.PriceObservationInput{
		ListingID: "l1", ObservedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Source: "source", Stock: domain.StockUnknown,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, id    string
		observation domain.PriceObservation
	}{
		{"empty result ID", "", valid}, {"blank result ID", " \t", valid},
		{"invalid observation", "result-1", domain.PriceObservation{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := store.InsertPriceObservation(ctx, test.id, test.observation); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
	if _, err := store.ListPriceObservations(ctx, " "); err == nil {
		t.Fatal("blank listing ID accepted")
	}
}
