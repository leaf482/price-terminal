package persistence_test

import (
	"context"
	"testing"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

// A nil pool proves invalid inputs are rejected before any SQL is attempted.
func TestValidationBeforeDatabaseAccess(t *testing.T) {
	store := persistence.New(nil)
	ctx := context.Background()
	for _, test := range []struct {
		name string
		run  func() error
	}{
		{"product", func() error { return store.InsertProduct(ctx, domain.Product{ID: " "}) }},
		{"retailer", func() error { return store.InsertRetailer(ctx, domain.Retailer{}) }},
		{"listing references", func() error { return store.InsertListing(ctx, domain.Listing{ID: "l1"}) }},
		{"listing URL", func() error {
			return store.InsertListing(ctx, domain.Listing{ID: "l1", ProductID: "p1", RetailerID: "r1", URL: "/relative"})
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); err == nil {
				t.Fatal("invalid domain value was accepted")
			}
		})
	}
}
