package persistence_test

import (
	"context"
	"testing"

	"github.com/leaf482/price-terminal/backend/persistence"
)

func TestCatalogListValidationBeforeSQL(t *testing.T) {
	store := persistence.New(nil)
	for _, limit := range []int{-1, 0, 101} {
		if _, err := store.ListRetailers(context.Background(), limit); err == nil {
			t.Fatalf("retailers accepted %d", limit)
		}
		if _, err := store.ListListingsByProduct(context.Background(), "p", limit); err == nil {
			t.Fatalf("listings accepted %d", limit)
		}
	}
	if _, err := store.ListListingsByProduct(context.Background(), " ", 1); err == nil {
		t.Fatal("blank product accepted")
	}
}
