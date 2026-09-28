package domain_test

import (
	"testing"

	"github.com/leaf482/price-terminal/backend/domain"
)

func TestCatalogValidation(t *testing.T) {
	for _, test := range []struct {
		name  string
		value interface{ Validate() error }
		valid bool
	}{
		{"product", domain.Product{ID: "p1", Name: "Item", Brand: "Brand", Model: "Model"}, true},
		{"product descriptions optional", domain.Product{ID: "p1"}, true},
		{"product missing ID", domain.Product{Name: "Item"}, false},
		{"product blank ID", domain.Product{ID: " \t"}, false},
		{"retailer", domain.Retailer{ID: "r1", Name: "Store"}, true},
		{"retailer name optional", domain.Retailer{ID: "r1"}, true},
		{"retailer missing ID", domain.Retailer{}, false},
		{"retailer blank ID", domain.Retailer{ID: "\n"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.value.Validate(); (err == nil) != test.valid {
				t.Errorf("Validate error = %v, want valid=%v", err, test.valid)
			}
		})
	}
}

func TestListingValidation(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*domain.Listing)
		valid  bool
	}{
		{"valid", func(*domain.Listing) {}, true},
		{"HTTP page", func(l *domain.Listing) { l.URL = "http://example.com/item" }, true},
		{"optional retailer reference", func(l *domain.Listing) { l.RetailerProductID = "" }, true},
		{"missing ID", func(l *domain.Listing) { l.ID = "" }, false},
		{"blank ID", func(l *domain.Listing) { l.ID = " " }, false},
		{"missing product", func(l *domain.Listing) { l.ProductID = "" }, false},
		{"blank product", func(l *domain.Listing) { l.ProductID = "\t" }, false},
		{"missing retailer", func(l *domain.Listing) { l.RetailerID = "" }, false},
		{"blank retailer", func(l *domain.Listing) { l.RetailerID = " " }, false},
		{"missing URL", func(l *domain.Listing) { l.URL = "" }, false},
		{"blank URL", func(l *domain.Listing) { l.URL = " " }, false},
		{"relative URL", func(l *domain.Listing) { l.URL = "/item" }, false},
		{"missing host", func(l *domain.Listing) { l.URL = "https:///item" }, false},
		{"non-page scheme", func(l *domain.Listing) { l.URL = "ftp://example.com/item" }, false},
		{"malformed URL", func(l *domain.Listing) { l.URL = "https://[" }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			listing := domain.Listing{ID: "l1", ProductID: "p1", RetailerID: "r1", URL: "https://example.com/item?variant=blue", RetailerProductID: "blue"}
			test.change(&listing)
			before := listing
			if err := listing.Validate(); (err == nil) != test.valid {
				t.Errorf("Validate error = %v, want valid=%v", err, test.valid)
			}
			if listing != before {
				t.Error("validation must not rewrite source identity")
			}
		})
	}
}
