//go:build integration

package persistence_test

import (
	"context"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"testing"
	"time"
)

func TestSearchIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := persistence.New(db)
	p := domain.Product{ID: "constructor", Name: "Camera NeedleName", Brand: "NeedleBrand", Model: "NeedleModel", Archived: true}
	r := domain.Retailer{ID: "__proto__", Name: "NeedleShop"}
	l := domain.Listing{ID: "toString", ProductID: p.ID, RetailerID: r.ID, URL: "https://example.com/NeedleURL?x=%25_", RetailerProductID: "NeedleSKU %_'\\", TrackingDisabled: true}
	for _, err := range []error{s.InsertProduct(ctx, p), s.InsertRetailer(ctx, r), s.InsertListing(ctx, l)} {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct{ q, kind string }{{"CONSTRUCTOR", "p"}, {"needlename", "p"}, {"needlebrand", "p"}, {"needlemodel", "p"}, {"__proto__", "r"}, {"needleshop", "r"}, {"TOSTRING", "l"}, {"needleurl", "l"}, {"needlesku", "l"}, {"%_'\\", "l"}, {"' OR 1=1 --", "none"}} {
		t.Run(tc.q, func(t *testing.T) {
			v, err := s.SearchCatalog(ctx, tc.q)
			if err != nil {
				t.Fatal(err)
			}
			switch tc.kind {
			case "p":
				if len(v.Products) != 1 || v.Products[0] != p {
					t.Fatal(v)
				}
			case "r":
				if len(v.Retailers) != 1 || v.Retailers[0] != r {
					t.Fatal(v)
				}
			case "l":
				if len(v.Listings) != 1 || v.Listings[0] != l {
					t.Fatal(v)
				}
			case "none":
				if len(v.Products)+len(v.Retailers)+len(v.Listings) != 0 {
					t.Fatal(v)
				}
			}
		})
	}
	for i := 24; i >= 0; i-- {
		id := fmt.Sprintf("bound-%02d", i)
		for _, err := range []error{s.InsertProduct(ctx, domain.Product{ID: id}), s.InsertRetailer(ctx, domain.Retailer{ID: id}), s.InsertListing(ctx, domain.Listing{ID: id, ProductID: p.ID, RetailerID: r.ID, URL: "https://example.com/" + id})} {
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	v, err := s.SearchCatalog(ctx, "bound-")
	if err != nil || len(v.Products) != 20 || len(v.Retailers) != 20 || len(v.Listings) != 20 {
		t.Fatal(v, err)
	}
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("bound-%02d", i)
		if v.Products[i].ID != id || v.Retailers[i].ID != id || v.Listings[i].ID != id {
			t.Fatal("ordering", v)
		}
	}
}
