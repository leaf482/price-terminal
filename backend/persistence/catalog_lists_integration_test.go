//go:build integration

package persistence_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

func TestCatalogListsIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	store := persistence.New(freshDatabase(t, ctx))
	if got, err := store.ListRetailers(ctx, 1); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty retailers=%v %v", got, err)
	}
	for _, id := range []string{"p", "other", "empty"} {
		if err := store.InsertProduct(ctx, domain.Product{ID: id}); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"empty", "missing"} {
		if got, err := store.ListListingsByProduct(ctx, id, 1); err != nil || got == nil || len(got) != 0 {
			t.Fatalf("empty listings=%v %v", got, err)
		}
	}
	makeListing := func(i int) domain.Listing {
		sku := ""
		if i%2 == 0 {
			sku = " SKU "
		}
		return domain.Listing{ID: fmt.Sprintf("l%03d", i), ProductID: "p", RetailerID: fmt.Sprintf("r%03d", i), URL: fmt.Sprintf("https://EXAMPLE.com/item%%2F%d?b=2&a=1#variant", i), RetailerProductID: sku}
	}
	for i := 104; i >= 0; i-- {
		if err := store.InsertRetailer(ctx, domain.Retailer{ID: fmt.Sprintf("r%03d", i), Name: "Store"}); err != nil {
			t.Fatal(err)
		}
		if err := store.InsertListing(ctx, makeListing(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.InsertListing(ctx, domain.Listing{ID: "aaa-other", ProductID: "other", RetailerID: "r000", URL: "https://example.com/other"}); err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{1, 50, 100} {
		retailers, err := store.ListRetailers(ctx, limit)
		if err != nil || len(retailers) != limit {
			t.Fatalf("retailers limit=%d count=%d err=%v", limit, len(retailers), err)
		}
		listings, err := store.ListListingsByProduct(ctx, "p", limit)
		if err != nil || len(listings) != limit {
			t.Fatalf("listings limit=%d count=%d err=%v", limit, len(listings), err)
		}
		for i := 0; i < limit; i++ {
			if retailers[i] != (domain.Retailer{ID: fmt.Sprintf("r%03d", i), Name: "Store"}) {
				t.Fatalf("retailer order/value=%+v", retailers[i])
			}
			if listings[i] != makeListing(i) {
				t.Fatalf("listing order/value=%+v", listings[i])
			}
		}
	}
}
