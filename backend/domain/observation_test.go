package domain_test

import (
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
)

func observationInput() domain.PriceObservationInput {
	return domain.PriceObservationInput{
		ListingID:  "l1",
		ObservedAt: time.Date(2026, 9, 28, 12, 30, 0, 0, time.FixedZone("source", 9*60*60)),
		Source:     "https://example.com/item",
		Stock:      domain.StockUnknown,
	}
}

func TestStockState(t *testing.T) {
	for _, test := range []struct {
		state domain.StockState
		valid bool
	}{
		{domain.StockUnknown, true},
		{domain.StockInStock, true},
		{domain.StockOutOfStock, true},
		{"", false},
		{"in stock", false},
		{"IN_STOCK", false},
		{"invalid", false},
	} {
		t.Run(string(test.state), func(t *testing.T) {
			if err := test.state.Validate(); (err == nil) != test.valid {
				t.Errorf("Validate error = %v, want valid=%v", err, test.valid)
			}
			input := observationInput()
			input.Stock = test.state
			_, err := domain.NewPriceObservation(input)
			if (err == nil) != test.valid {
				t.Errorf("observation error = %v, want valid=%v", err, test.valid)
			}
		})
	}
}

func TestObservationValidation(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*domain.PriceObservationInput)
		valid  bool
	}{
		{"stock only", func(*domain.PriceObservationInput) {}, true},
		{"missing listing", func(i *domain.PriceObservationInput) { i.ListingID = "" }, false},
		{"blank listing", func(i *domain.PriceObservationInput) { i.ListingID = " " }, false},
		{"missing time", func(i *domain.PriceObservationInput) { i.ObservedAt = time.Time{} }, false},
		{"missing source", func(i *domain.PriceObservationInput) { i.Source = "" }, false},
		{"blank source", func(i *domain.PriceObservationInput) { i.Source = "\t" }, false},
		{"prices mix currencies", func(i *domain.PriceObservationInput) {
			i.SalePrice = &domain.Money{MinorUnits: 100, Currency: domain.USD}
			i.OfferPrice = &domain.Money{MinorUnits: 100, Currency: domain.JPY}
		}, false},
		{"missing stock", func(i *domain.PriceObservationInput) { i.Stock = "" }, false},
		{"MSRP without evidence", func(i *domain.PriceObservationInput) { i.MSRP = &domain.Money{MinorUnits: 100, Currency: domain.USD} }, false},
		{"MSRP blank evidence", func(i *domain.PriceObservationInput) {
			i.MSRP = &domain.Money{MinorUnits: 100, Currency: domain.USD}
			i.MSRPSource = " "
		}, false},
		{"MSRP evidence without value", func(i *domain.PriceObservationInput) { i.MSRPSource = "manufacturer page" }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := observationInput()
			test.change(&input)
			observation, err := domain.NewPriceObservation(input)
			if (err == nil) != test.valid {
				t.Fatalf("NewPriceObservation error = %v, want valid=%v", err, test.valid)
			}
			if test.valid {
				if err := observation.Validate(); err != nil {
					t.Fatal(err)
				}
				if observation.ListingID() != input.ListingID || observation.Source() != input.Source || observation.Stock() != input.Stock {
					t.Error("observation facts changed")
				}
				if currency, present := observation.Currency(); present || currency != "" {
					t.Errorf("stock-only currency = %q, %v; want empty, false", currency, present)
				}
				if !observation.ObservedAt().Equal(input.ObservedAt) || observation.ObservedAt().Location() != time.UTC {
					t.Error("observation time must preserve the instant in UTC")
				}
			}
		})
	}
	if err := (domain.PriceObservation{}).Validate(); err == nil {
		t.Error("zero observation must not be valid")
	}
}

// Check each price role independently so none can skip validation or overwrite
// another role. These functions exercise only the package's public API.
var priceRoles = []struct {
	name string
	set  func(*domain.PriceObservationInput, *domain.Money)
	get  func(domain.PriceObservation) (domain.Money, bool)
}{
	{"MSRP", func(i *domain.PriceObservationInput, p *domain.Money) {
		i.MSRP = p
		if p != nil {
			i.MSRPSource = "explicit MSRP on manufacturer page"
		}
	}, domain.PriceObservation.MSRP},
	{"retailer list", func(i *domain.PriceObservationInput, p *domain.Money) { i.RetailerListPrice = p }, domain.PriceObservation.RetailerListPrice},
	{"sale", func(i *domain.PriceObservationInput, p *domain.Money) { i.SalePrice = p }, domain.PriceObservation.SalePrice},
	{"offer", func(i *domain.PriceObservationInput, p *domain.Money) { i.OfferPrice = p }, domain.PriceObservation.OfferPrice},
}

func TestPriceRoleInvariants(t *testing.T) {
	for _, role := range priceRoles {
		t.Run(role.name, func(t *testing.T) {
			for _, test := range []struct {
				name  string
				price *domain.Money
				valid bool
			}{
				{"missing", nil, true},
				{"zero", &domain.Money{Currency: domain.USD}, true},
				{"positive", &domain.Money{MinorUnits: 500, Currency: domain.USD}, true},
				{"JPY price", &domain.Money{MinorUnits: 500, Currency: domain.JPY}, true},
				{"JPY zero", &domain.Money{Currency: domain.JPY}, true},
				{"negative", &domain.Money{MinorUnits: -1, Currency: domain.USD}, false},
				{"zero-value Money", &domain.Money{}, false},
				{"unsupported currency", &domain.Money{Currency: "EUR"}, false},
			} {
				t.Run(test.name, func(t *testing.T) {
					input := observationInput()
					role.set(&input, test.price)
					observation, err := domain.NewPriceObservation(input)
					if (err == nil) != test.valid {
						t.Fatalf("error = %v, want valid=%v", err, test.valid)
					}
					if !test.valid {
						return
					}
					currency, present := observation.Currency()
					if test.price == nil {
						if present || currency != "" {
							t.Errorf("missing prices gave currency %q, %v", currency, present)
						}
					} else if !present || currency != test.price.Currency {
						t.Errorf("currency = %q, %v; want %q, true", currency, present, test.price.Currency)
					}
					for _, other := range priceRoles {
						value, present := other.get(observation)
						wantPresent := other.name == role.name && test.price != nil
						if present != wantPresent || (wantPresent && value != *test.price) {
							t.Errorf("%s: price=%+v present=%v", other.name, value, present)
						}
					}
				})
			}
		})
	}
}

func TestSeparatePricesAndImmutability(t *testing.T) {
	for _, currency := range []domain.Currency{domain.USD, domain.JPY} {
		t.Run(string(currency), func(t *testing.T) {
			input := observationInput()
			prices := []domain.Money{
				{MinorUnits: 12000, Currency: currency},
				{MinorUnits: 11000, Currency: currency},
				{MinorUnits: 9000, Currency: currency},
				{MinorUnits: 9500, Currency: currency},
			}
			for index, role := range priceRoles {
				role.set(&input, &prices[index])
			}
			observation, err := domain.NewPriceObservation(input)
			if err != nil {
				t.Fatal(err)
			}
			if observation.MSRPSource() != input.MSRPSource {
				t.Error("MSRP provenance lost")
			}
			input.ListingID = "changed"
			input.Source = "changed"
			input.MSRPSource = "changed"
			for index, role := range priceRoles {
				want := prices[index]
				prices[index] = domain.Money{}
				value, present := role.get(observation)
				if !present || value != want {
					t.Errorf("%s was changed through its input pointer", role.name)
				}
				value.MinorUnits = -1
				again, _ := role.get(observation)
				if again != want {
					t.Errorf("%s was changed through its returned value", role.name)
				}
			}
			if observation.ListingID() != "l1" || observation.Source() != observationInput().Source || observation.MSRPSource() == "changed" {
				t.Error("observation changed when input was changed")
			}
			if err := observation.Validate(); err != nil {
				t.Fatal(err)
			}
			if got, present := observation.Currency(); !present || got != currency {
				t.Errorf("currency after input mutation = %q, %v; want %q, true", got, present, currency)
			}
		})
	}
}
