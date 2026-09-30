package domain

import (
	"math"
	"testing"
	"time"
)

func alertInt(n int64) *int64 { return &n }
func TestAlertConfiguration(t *testing.T) {
	base := PriceAlert{ID: "a", ListingID: "l", Kind: "target", Currency: USD, Threshold: alertInt(0), Enabled: true}
	for _, tt := range []struct {
		name   string
		change func(*PriceAlert)
		valid  bool
	}{
		{"zero", func(a *PriceAlert) {}, true},
		{"drop", func(a *PriceAlert) { a.Kind = "drop"; a.Threshold = nil; a.DropBasisPoints = alertInt(1) }, true},
		{"low", func(a *PriceAlert) { a.Kind = "historical_low"; a.Threshold = nil }, true},
		{"id", func(a *PriceAlert) { a.ID = " " }, false},
		{"listing", func(a *PriceAlert) { a.ListingID = "" }, false},
		{"currency", func(a *PriceAlert) { a.Currency = "EUR" }, false},
		{"kind", func(a *PriceAlert) { a.Kind = "unknown" }, false},
		{"missing threshold", func(a *PriceAlert) { a.Threshold = nil }, false},
		{"negative", func(a *PriceAlert) { a.Threshold = alertInt(-1) }, false},
		{"extra parameter", func(a *PriceAlert) { a.DropBasisPoints = alertInt(10) }, false},
		{"zero drop", func(a *PriceAlert) { a.Kind = "drop"; a.Threshold = nil; a.DropBasisPoints = alertInt(0) }, false},
		{"large drop", func(a *PriceAlert) { a.Kind = "drop"; a.Threshold = nil; a.DropBasisPoints = alertInt(10001) }, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := base
			tt.change(&a)
			if (a.Validate() == nil) != tt.valid {
				t.Fatal(a.Validate())
			}
		})
	}
}

func TestAlertMatches(t *testing.T) {
	usd := func(n int64) *Money { return &Money{MinorUnits: n, Currency: USD} }
	for _, tt := range []struct {
		name, kind            string
		current, prior        *Money
		stock                 StockState
		enabled, require, hit bool
	}{
		{"target hit", "target", usd(90), nil, StockInStock, true, true, true},
		{"target equal", "target", usd(100), nil, StockInStock, true, true, true},
		{"target miss", "target", usd(101), nil, StockInStock, true, true, false},
		{"zero", "target", usd(0), nil, StockInStock, true, true, true},
		{"drop hit", "drop", usd(90), usd(100), StockInStock, true, true, true},
		{"drop miss", "drop", usd(91), usd(100), StockInStock, true, true, false},
		{"drop first", "drop", usd(1), nil, StockInStock, true, true, false},
		{"zero denominator", "drop", usd(0), usd(0), StockInStock, true, true, false},
		{"large exact", "drop", usd(0), usd(math.MaxInt64), StockInStock, true, true, true},
		{"low hit", "historical_low", usd(90), usd(91), StockInStock, true, true, true},
		{"low equal", "historical_low", usd(90), usd(90), StockInStock, true, true, false},
		{"low first", "historical_low", usd(90), nil, StockInStock, true, true, false},
		{"prior currency", "drop", usd(1), &Money{100, JPY}, StockInStock, true, true, false},
		{"current currency", "target", &Money{1, JPY}, nil, StockInStock, true, true, false},
		{"stock only", "target", nil, nil, StockInStock, true, true, false},
		{"disabled", "target", usd(1), nil, StockInStock, false, true, false},
		{"stock required", "target", usd(1), nil, StockUnknown, true, true, false},
		{"stock optional", "target", usd(1), nil, StockOutOfStock, true, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := PriceAlert{ID: "a", ListingID: "l", Kind: tt.kind, Currency: USD, Enabled: tt.enabled, RequireInStock: tt.require}
			if a.Kind == "target" {
				a.Threshold = alertInt(100)
			}
			if a.Kind == "drop" {
				a.DropBasisPoints = alertInt(1000)
			}
			o, err := NewPriceObservation(PriceObservationInput{ListingID: "l", ObservedAt: time.Unix(100, 0), Source: "fixture", Stock: tt.stock, OfferPrice: tt.current})
			if err != nil {
				t.Fatal(err)
			}
			got, err := a.Matches(o, tt.prior, tt.prior)
			if err != nil || got != tt.hit {
				t.Fatalf("hit=%v err=%v", got, err)
			}
		})
	}
}

func TestAlertPriceBasis(t *testing.T) {
	offer, sale, list := Money{100, USD}, Money{50, USD}, Money{200, USD}
	in := PriceObservationInput{ListingID: "l", ObservedAt: time.Unix(100, 0), Source: "fixture", Stock: StockInStock, OfferPrice: &offer, SalePrice: &sale, RetailerListPrice: &list}
	for _, tt := range []struct {
		offer, sale *Money
		want        int64
		present     bool
	}{{&offer, &sale, 100, true}, {nil, &sale, 50, true}, {nil, nil, 0, false}} {
		in.OfferPrice, in.SalePrice = tt.offer, tt.sale
		o, err := NewPriceObservation(in)
		if err != nil {
			t.Fatal(err)
		}
		m, ok := AlertPrice(o)
		if ok != tt.present || m.MinorUnits != tt.want {
			t.Fatal(m, ok)
		}
	}
}
