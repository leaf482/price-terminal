package domain

import (
	"fmt"
	"strings"
	"time"
)

type StockState string

const (
	StockUnknown    StockState = "unknown"
	StockInStock    StockState = "in_stock"
	StockOutOfStock StockState = "out_of_stock"
)

func (s StockState) Validate() error {
	switch s {
	case StockUnknown, StockInStock, StockOutOfStock:
		return nil
	default:
		return fmt.Errorf("invalid stock state %q", s)
	}
}

// PriceObservationInput supplies source claims, not calculated prices. Nil price
// pointers mean absent; a pointer to valid zero Money means explicitly observed
// zero. Source identifies the evidence for this observation. MSRPSource must
// identify an explicit manufacturer-suggested-price claim when MSRP is supplied;
// the caller is responsible for that evidence, not inferring it from a list price.
type PriceObservationInput struct {
	ListingID         string
	ObservedAt        time.Time
	Source            string
	Stock             StockState
	MSRP              *Money
	MSRPSource        string
	RetailerListPrice *Money
	SalePrice         *Money
	OfferPrice        *Money
}

type observedMoney struct {
	value   Money
	present bool
}

func copyPrice(price *Money) observedMoney {
	if price == nil {
		return observedMoney{}
	}
	return observedMoney{value: *price, present: true}
}

// PriceObservation is an immutable snapshot: all fields are private, constructor
// inputs are copied, and accessors return values rather than mutable pointers.
// No price precedence, discounts, ingestion time, or persistence identity is
// inferred. A valid stock-only observation may have all four prices absent.
type PriceObservation struct {
	listingID         string
	observedAt        time.Time
	source            string
	stock             StockState
	msrp              observedMoney
	msrpSource        string
	retailerListPrice observedMoney
	salePrice         observedMoney
	offerPrice        observedMoney
}

func NewPriceObservation(input PriceObservationInput) (PriceObservation, error) {
	observation := PriceObservation{
		listingID:         input.ListingID,
		observedAt:        input.ObservedAt.UTC(),
		source:            input.Source,
		stock:             input.Stock,
		msrp:              copyPrice(input.MSRP),
		msrpSource:        input.MSRPSource,
		retailerListPrice: copyPrice(input.RetailerListPrice),
		salePrice:         copyPrice(input.SalePrice),
		offerPrice:        copyPrice(input.OfferPrice),
	}
	if err := observation.Validate(); err != nil {
		return PriceObservation{}, err
	}
	return observation, nil
}

func (o PriceObservation) Validate() error {
	if err := requireText("listing ID", o.listingID); err != nil {
		return err
	}
	if o.observedAt.IsZero() {
		return fmt.Errorf("observation time is required")
	}
	if err := requireText("observation source", o.source); err != nil {
		return err
	}
	if err := o.stock.Validate(); err != nil {
		return err
	}
	if o.msrp.present {
		if err := requireText("MSRP source", o.msrpSource); err != nil {
			return err
		}
	} else if strings.TrimSpace(o.msrpSource) != "" {
		return fmt.Errorf("MSRP source requires an MSRP value")
	}
	currency, _ := o.Currency()
	for _, price := range []struct {
		name  string
		price observedMoney
	}{
		{"MSRP", o.msrp},
		{"retailer list price", o.retailerListPrice},
		{"sale price", o.salePrice},
		{"offer price", o.offerPrice},
	} {
		if !price.price.present {
			continue
		}
		if err := price.price.value.Validate(); err != nil {
			return fmt.Errorf("%s: %w", price.name, err)
		}
		if price.price.value.Currency != currency {
			return fmt.Errorf("%s currency does not match other observed prices", price.name)
		}
	}
	return nil
}

func (o PriceObservation) ListingID() string     { return o.listingID }
func (o PriceObservation) ObservedAt() time.Time { return o.observedAt }
func (o PriceObservation) Source() string        { return o.source }
func (o PriceObservation) Stock() StockState     { return o.stock }
func (o PriceObservation) MSRPSource() string    { return o.msrpSource }

// Currency derives the shared currency from present prices. A stock-only
// observation returns ("", false); an explicit zero price establishes currency.
// Validate ensures all present prices use the same supported currency.
func (o PriceObservation) Currency() (Currency, bool) {
	for _, price := range [...]observedMoney{o.msrp, o.retailerListPrice, o.salePrice, o.offerPrice} {
		if price.present {
			return price.value.Currency, true
		}
	}
	return "", false
}

// Price accessors return a presence flag; ignore the Money value when false.
func (o PriceObservation) MSRP() (Money, bool) {
	return o.msrp.value, o.msrp.present
}

func (o PriceObservation) RetailerListPrice() (Money, bool) {
	return o.retailerListPrice.value, o.retailerListPrice.present
}

func (o PriceObservation) SalePrice() (Money, bool) {
	return o.salePrice.value, o.salePrice.present
}

func (o PriceObservation) OfferPrice() (Money, bool) {
	return o.offerPrice.value, o.offerPrice.present
}
