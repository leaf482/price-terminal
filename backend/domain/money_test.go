package domain_test

import (
	"math"
	"testing"

	"github.com/leaf482/price-terminal/backend/domain"
)

func TestCurrency(t *testing.T) {
	for _, test := range []struct {
		code     domain.Currency
		exponent uint8
		valid    bool
	}{
		{domain.USD, 2, true},
		{domain.JPY, 0, true},
		{"", 0, false},
		{"usd", 0, false},
		{" USD ", 0, false},
		{"EUR", 0, false},
		{"invalid", 0, false},
	} {
		t.Run(string(test.code), func(t *testing.T) {
			exponent, err := test.code.MinorUnitExponent()
			if (err == nil) != test.valid || (test.code.Validate() == nil) != test.valid {
				t.Fatalf("currency validity differs from %v: %v", test.valid, err)
			}
			if test.valid && exponent != test.exponent {
				t.Errorf("exponent = %d, want %d", exponent, test.exponent)
			}
		})
	}
}

func TestMoney(t *testing.T) {
	for _, test := range []struct {
		name     string
		amount   int64
		currency domain.Currency
		valid    bool
	}{
		{"USD cents", 1999, domain.USD, true},
		{"JPY units", 1999, domain.JPY, true},
		{"explicit zero", 0, domain.USD, true},
		{"maximum int64", math.MaxInt64, domain.USD, true},
		{"negative", -1, domain.USD, false},
		{"minimum int64", math.MinInt64, domain.USD, false},
		{"missing currency", 0, "", false},
		{"unsupported currency", 100, "EUR", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			money, err := domain.NewMoney(test.amount, test.currency)
			if (err == nil) != test.valid {
				t.Fatalf("NewMoney error = %v, want valid=%v", err, test.valid)
			}
			raw := domain.Money{MinorUnits: test.amount, Currency: test.currency}
			if (raw.Validate() == nil) != test.valid {
				t.Fatal("literal validation and constructor disagree")
			}
			if test.valid && money != raw {
				t.Errorf("amount or currency changed: got %+v, want %+v", money, raw)
			}
		})
	}
}
