// Package domain defines pure product and observed-price values. Constructors
// and Validate methods reject invalid values; persistence and collection are
// outside this package. Go zero values must be validated before use.
package domain

import "fmt"

// Currency is an explicitly supported currency code, not an arbitrary label.
type Currency string

const (
	USD Currency = "USD"
	JPY Currency = "JPY"
)

// MinorUnitExponent describes the currency's scale without assuming two places.
// Only USD and JPY are supported initially. No conversion or arithmetic is done.
func (c Currency) MinorUnitExponent() (uint8, error) {
	switch c {
	case USD:
		return 2, nil
	case JPY:
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported currency %q", c)
	}
}

func (c Currency) Validate() error {
	_, err := c.MinorUnitExponent()
	return err
}

// Money stores an explicitly supplied amount, including zero. Missing prices
// must be represented separately; the zero Money value has no valid currency.
type Money struct {
	MinorUnits int64
	Currency   Currency
}

func NewMoney(minorUnits int64, currency Currency) (Money, error) {
	money := Money{MinorUnits: minorUnits, Currency: currency}
	if err := money.Validate(); err != nil {
		return Money{}, err
	}
	return money, nil
}

func (m Money) Validate() error {
	if err := m.Currency.Validate(); err != nil {
		return err
	}
	if m.MinorUnits < 0 {
		return fmt.Errorf("monetary amount must not be negative")
	}
	return nil
}
