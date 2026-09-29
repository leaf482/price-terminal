package provider_test

import (
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/provider"
)

func listing() domain.Listing {
	return domain.Listing{ID: "l1", ProductID: "p1", RetailerID: "r1",
		URL: "https://example.com/item?variant=blue", RetailerProductID: "blue"}
}

func facts() domain.PriceObservationInput {
	return domain.PriceObservationInput{ListingID: "l1",
		ObservedAt: time.Date(2026, 9, 28, 12, 0, 0, 123, time.UTC),
		Source:     "https://example.com/item?variant=blue", Stock: domain.StockUnknown}
}

func TestSuccessfulResults(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*domain.PriceObservationInput)
	}{
		{"stock only", func(i *domain.PriceObservationInput) { i.Stock = domain.StockOutOfStock }},
		{"unknown stock and unavailable prices", func(*domain.PriceObservationInput) {}},
		{"explicit zero USD offer", func(i *domain.PriceObservationInput) { i.OfferPrice = &domain.Money{Currency: domain.USD} }},
		{"JPY offer with other prices missing", func(i *domain.PriceObservationInput) {
			i.OfferPrice = &domain.Money{MinorUnits: 500, Currency: domain.JPY}
		}},
		{"all price meanings", func(i *domain.PriceObservationInput) {
			i.Stock = domain.StockInStock
			i.MSRP = &domain.Money{MinorUnits: 12000, Currency: domain.USD}
			i.MSRPSource = "explicit manufacturer MSRP claim"
			i.RetailerListPrice = &domain.Money{MinorUnits: 11000, Currency: domain.USD}
			i.SalePrice = &domain.Money{MinorUnits: 9000, Currency: domain.USD}
			i.OfferPrice = &domain.Money{MinorUnits: 9500, Currency: domain.USD}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := facts()
			test.change(&input)
			result, err := domain.NewPriceObservation(input)
			if err != nil {
				t.Fatal(err)
			}
			if err := provider.ValidateResult(listing(), result); err != nil {
				t.Fatal(err)
			}
			if result.Source() != input.Source || result.MSRPSource() != input.MSRPSource || result.Stock() != input.Stock || !result.ObservedAt().Equal(input.ObservedAt) {
				t.Fatal("source facts changed")
			}
			var expectedCurrency domain.Currency
			for _, price := range []struct {
				want *domain.Money
				get  func() (domain.Money, bool)
			}{{input.MSRP, result.MSRP}, {input.RetailerListPrice, result.RetailerListPrice}, {input.SalePrice, result.SalePrice}, {input.OfferPrice, result.OfferPrice}} {
				got, present := price.get()
				if present != (price.want != nil) || (present && got != *price.want) {
					t.Fatal("price meaning or presence changed")
				}
				if price.want != nil {
					expectedCurrency = price.want.Currency
				}
			}
			currency, present := result.Currency()
			if currency != expectedCurrency || present != (expectedCurrency != "") {
				t.Fatal("currency presence changed")
			}
		})
	}
}

func TestInvalidNormalizedFacts(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*domain.PriceObservationInput)
	}{
		{"missing time", func(i *domain.PriceObservationInput) { i.ObservedAt = time.Time{} }},
		{"missing source", func(i *domain.PriceObservationInput) { i.Source = "" }},
		{"missing listing", func(i *domain.PriceObservationInput) { i.ListingID = "" }},
		{"invalid stock", func(i *domain.PriceObservationInput) { i.Stock = "invalid" }},
		{"unsupported currency", func(i *domain.PriceObservationInput) { i.OfferPrice = &domain.Money{Currency: "EUR"} }},
		{"negative amount", func(i *domain.PriceObservationInput) {
			i.OfferPrice = &domain.Money{MinorUnits: -1, Currency: domain.USD}
		}},
		{"mixed currencies", func(i *domain.PriceObservationInput) {
			i.SalePrice = &domain.Money{MinorUnits: 1, Currency: domain.USD}
			i.OfferPrice = &domain.Money{MinorUnits: 1, Currency: domain.JPY}
		}},
		{"MSRP without evidence", func(i *domain.PriceObservationInput) { i.MSRP = &domain.Money{Currency: domain.USD} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := facts()
			test.change(&input)
			result, err := domain.NewPriceObservation(input)
			if err == nil {
				t.Fatal("malformed normalized facts became a valid observation")
			}
			if err := provider.ValidateResult(listing(), result); err == nil {
				t.Fatal("invalid result accepted")
			}
		})
	}
}

func TestResultListingIdentity(t *testing.T) {
	result, err := domain.NewPriceObservation(facts())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		change func(*domain.Listing)
	}{
		{"different listing", func(l *domain.Listing) { l.ID = "l2" }},
		{"missing ID", func(l *domain.Listing) { l.ID = "" }},
		{"missing product", func(l *domain.Listing) { l.ProductID = "" }},
		{"missing retailer", func(l *domain.Listing) { l.RetailerID = "" }},
		{"invalid source URL", func(l *domain.Listing) { l.URL = "/relative" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := listing()
			test.change(&request)
			if err := provider.ValidateResult(request, result); err == nil {
				t.Fatal("invalid listing or identity mismatch accepted")
			}
		})
	}
	request := listing()
	request.RetailerProductID = ""
	if err := provider.ValidateResult(request, result); err != nil {
		t.Fatalf("optional retailer product ID required: %v", err)
	}
	if err := provider.ValidateResult(listing(), domain.PriceObservation{}); err == nil {
		t.Fatal("zero result accepted")
	}
}
