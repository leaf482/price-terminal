package persistence

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
)

type scannerFunc func(...any) error

func (f scannerFunc) Scan(dst ...any) error { return f(dst...) }

func TestCatalogScanners(t *testing.T) {
	for _, id := range []string{"constructor", "__proto__", "toString", ""} {
		t.Run(id, func(t *testing.T) {
			p, err := scanProduct(scannerFunc(func(d ...any) error {
				*d[0].(*string) = id
				*d[1].(*string) = " name "
				*d[2].(*string) = "brand"
				*d[3].(*string) = "model"
				*d[4].(*bool) = true
				return nil
			}))
			if (err != nil) != (id == "") {
				t.Fatal("product validation", err)
			}
			if id != "" && p != (domain.Product{ID: id, Name: " name ", Brand: "brand", Model: "model", Archived: true}) {
				t.Fatal(p)
			}
			r, err := scanRetailer(scannerFunc(func(d ...any) error { *d[0].(*string) = id; *d[1].(*string) = " retailer "; return nil }))
			if (err != nil) != (id == "") {
				t.Fatal("retailer validation", err)
			}
			if id != "" && r != (domain.Retailer{ID: id, Name: " retailer "}) {
				t.Fatal(r)
			}
			for _, disabled := range []bool{false, true} {
				l, err := scanListing(scannerFunc(func(d ...any) error {
					for i, v := range []string{id, "product", "retailer", "https://example.com/item?x=%25", ""} {
						*d[i].(*string) = v
					}
					*d[5].(*bool) = disabled
					return nil
				}))
				if (err != nil) != (id == "") {
					t.Fatal("listing validation", err)
				}
				if id != "" && l != (domain.Listing{ID: id, ProductID: "product", RetailerID: "retailer", URL: "https://example.com/item?x=%25", TrackingDisabled: disabled}) {
					t.Fatal(l)
				}
			}
		})
	}
	for _, cause := range []error{sql.ErrNoRows, errors.New("scan conversion failed")} {
		row := scannerFunc(func(...any) error { return cause })
		_, p := scanProduct(row)
		_, r := scanRetailer(row)
		_, l := scanListing(row)
		for _, err := range []error{p, r, l} {
			if !errors.Is(err, cause) {
				t.Fatal("scan error lost", err)
			}
		}
	}
}

func TestObservationAmounts(t *testing.T) {
	at := time.Date(2026, 10, 3, 1, 2, 3, 123456789, time.FixedZone("offset", 9*3600))
	for _, currency := range []domain.Currency{domain.USD, domain.JPY} {
		for role := 0; role < 4; role++ {
			input := domain.PriceObservationInput{ListingID: "constructor", ObservedAt: at, Source: "CSV evidence", Stock: domain.StockUnknown}
			if role == 0 {
				input.MSRPSource = "explicit MSRP evidence"
			}
			var amounts [4]sql.NullInt64
			amounts[role] = sql.NullInt64{Int64: 0, Valid: true}
			o, err := observationFromAmounts(input, sql.NullString{String: string(currency), Valid: true}, amounts)
			if err != nil {
				t.Fatal(err)
			}
			getters := []func() (domain.Money, bool){o.MSRP, o.RetailerListPrice, o.SalePrice, o.OfferPrice}
			for i, get := range getters {
				m, ok := get()
				if ok != (i == role) || ok && (m.MinorUnits != 0 || m.Currency != currency) {
					t.Fatal("role/zero changed", i, m, ok)
				}
			}
			if !o.ObservedAt().Equal(at) || o.ObservedAt().Nanosecond() != 123456789 || o.Source() != input.Source || o.MSRPSource() != input.MSRPSource {
				t.Fatal("facts changed")
			}
		}
	}
	input := domain.PriceObservationInput{ListingID: "l", ObservedAt: at, Source: "manual", Stock: domain.StockOutOfStock}
	o, err := observationFromAmounts(input, sql.NullString{}, [4]sql.NullInt64{})
	if err != nil {
		t.Fatal(err)
	}
	if _, present := o.Currency(); present {
		t.Fatal("invented stock-only currency")
	}
	for _, tt := range []struct {
		name, currency string
		amount         int64
		stock          domain.StockState
	}{
		{"no currency", "", 0, domain.StockUnknown},
		{"unsupported currency", "EUR", 0, domain.StockUnknown},
		{"negative", "USD", -1, domain.StockUnknown},
		{"bad stock", "USD", 1, "broken"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			in := input
			in.Stock = tt.stock
			var amounts [4]sql.NullInt64
			amounts[3] = sql.NullInt64{Int64: tt.amount, Valid: true}
			if _, err := observationFromAmounts(in, sql.NullString{String: tt.currency, Valid: tt.currency != ""}, amounts); err == nil {
				t.Fatal("malformed state accepted")
			}
		})
	}
	var msrp [4]sql.NullInt64
	msrp[0] = sql.NullInt64{Int64: 1, Valid: true}
	if _, err := observationFromAmounts(input, sql.NullString{String: "USD", Valid: true}, msrp); err == nil {
		t.Fatal("missing MSRP evidence accepted")
	}
}
