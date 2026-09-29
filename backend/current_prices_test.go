package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

type currentStub struct {
	values          []persistence.CurrentListing
	err, productErr error
}

func (s currentStub) GetProduct(context.Context, string) (domain.Product, error) {
	return domain.Product{ID: "p"}, s.productErr
}
func (s currentStub) CurrentListing(context.Context, string) (persistence.CurrentListing, error) {
	if s.err != nil {
		return persistence.CurrentListing{}, s.err
	}
	return s.values[0], nil
}
func (s currentStub) CurrentProduct(context.Context, string) ([]persistence.CurrentListing, error) {
	return s.values, s.err
}

var priceNow = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

func currentFixture(t *testing.T, id string, currency domain.Currency, amount *int64, at time.Time) persistence.CurrentListing {
	t.Helper()
	var price *domain.Money
	if amount != nil {
		price = &domain.Money{MinorUnits: *amount, Currency: currency}
	}
	o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: id, ObservedAt: at, Source: "fixture", Stock: domain.StockInStock, OfferPrice: price})
	if err != nil {
		t.Fatal(err)
	}
	return persistence.CurrentListing{Listing: domain.Listing{ID: id, ProductID: "p", RetailerID: "retailer-" + id, URL: "https://example.com/" + id}, Observation: &o}
}
func intPrice(n int64) *int64 { return &n }
func currentRequest(t *testing.T, s currentStub, path string, want int) map[string]json.RawMessage {
	t.Helper()
	api := currentAPI{store: s, status: func(string) collector.Status {
		return collector.Status{Active: true, State: "failed", Error: "collection_failed"}
	}, now: func() time.Time { return priceNow }, maxAge: time.Minute}
	w := httptest.NewRecorder()
	newHandler(nil, nil, nil, api).ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	if w.Code != want || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "secret") {
		t.Fatal("internal details leaked")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope
}
func TestListingCurrentAPI(t *testing.T) {
	for _, test := range []struct {
		name      string
		amount    *int64
		at        time.Time
		freshness string
	}{{"zero", intPrice(0), priceNow, "fresh"}, {"stock only", nil, priceNow, "fresh"}, {"old", intPrice(10), priceNow.Add(-time.Hour), "stale"}, {"future", intPrice(10), priceNow.Add(time.Hour), "future"}} {
		t.Run(test.name, func(t *testing.T) {
			value := currentFixture(t, "l", domain.USD, test.amount, test.at)
			envelope := currentRequest(t, currentStub{values: []persistence.CurrentListing{value}}, "/listings/l/price", 200)
			var got currentJSON
			if err := json.Unmarshal(envelope["data"], &got); err != nil {
				t.Fatal(err)
			}
			if got.Freshness != test.freshness || !got.Observation.ObservedAt.Equal(test.at) || got.Collection.State != "failed" {
				t.Fatal(got)
			}
			if test.amount == nil {
				if got.Observation.OfferPrice != nil || strings.Contains(string(envelope["data"]), "currency") {
					t.Fatal("stock-only fabricated price")
				}
			} else if got.Observation.OfferPrice == nil || *got.Observation.OfferPrice != *test.amount {
				t.Fatal("price lost")
			}
		})
	}
	v := currentFixture(t, "l", domain.USD, nil, priceNow)
	v.Observation = nil
	env := currentRequest(t, currentStub{values: []persistence.CurrentListing{v}}, "/listings/l/price", 200)
	if !strings.Contains(string(env["data"]), `"observation":null`) {
		t.Fatal("missing history")
	}
}
func TestComparableBest(t *testing.T) {
	base := func(id string, amount int64) currentJSON {
		return currentResponse(currentFixture(t, id, domain.USD, intPrice(amount), priceNow), collector.Status{}, priceNow, time.Minute)
	}
	for _, test := range []struct {
		name           string
		change         func([]currentJSON)
		reason, winner string
	}{
		{"lowest", func([]currentJSON) {}, "comparable", "b"},
		{"mixed", func(x []currentJSON) { x[1].Observation.Currency = domain.JPY }, "incompatible_currencies", ""},
		{"missing", func(x []currentJSON) { x[1].Observation = nil }, "missing_observation", ""},
		{"stock only", func(x []currentJSON) { x[1].Observation.OfferPrice = nil; x[1].Observation.Currency = "" }, "missing_price", ""},
		{"stale", func(x []currentJSON) { x[1].Freshness = "stale" }, "not_fresh", ""},
		{"unknown stock", func(x []currentJSON) { x[1].Observation.Stock = domain.StockUnknown }, "not_in_stock", ""},
		{"reference only", func(x []currentJSON) {
			x[1].Observation.OfferPrice = nil
			x[1].Observation.MSRP = intPrice(1)
			x[1].Observation.RetailerListPrice = intPrice(2)
		}, "missing_price", ""},
		{"sale fallback", func(x []currentJSON) { x[1].Observation.OfferPrice = nil; x[1].Observation.SalePrice = intPrice(5) }, "comparable", "b"},
		{"offer precedence", func(x []currentJSON) { x[0].Observation.SalePrice = intPrice(1) }, "comparable", "b"},
		{"tie", func(x []currentJSON) { x[1].Observation.OfferPrice = intPrice(20) }, "comparable", "a"},
		{"zero", func(x []currentJSON) { x[0].Observation.OfferPrice = intPrice(0) }, "comparable", "a"},
	} {
		t.Run(test.name, func(t *testing.T) {
			items := []currentJSON{base("a", 20), base("b", 10)}
			test.change(items)
			best, reason := comparableBest(items)
			if reason != test.reason {
				t.Fatal(reason)
			}
			if test.winner == "" {
				if best != nil {
					t.Fatal("false best")
				}
			} else if best == nil || best.ListingID != test.winner {
				t.Fatal(best)
			}
		})
	}
	if best, reason := comparableBest(nil); best != nil || reason != "no_listings" {
		t.Fatal("empty comparison")
	}
}
func TestProductCurrentAPI(t *testing.T) {
	s := currentStub{values: []persistence.CurrentListing{currentFixture(t, "a", domain.USD, intPrice(20), priceNow), currentFixture(t, "b", domain.USD, intPrice(10), priceNow)}}
	env := currentRequest(t, s, "/products/p/prices", 200)
	var got struct {
		Listings []currentJSON `json:"listings"`
		Best     *bestJSON     `json:"best_price"`
	}
	if err := json.Unmarshal(env["data"], &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Listings) != 2 || got.Best.ListingID != "b" || got.Listings[0].Listing.RetailerID == got.Listings[1].Listing.RetailerID {
		t.Fatal(got)
	}
	s.values[1] = currentFixture(t, "b", domain.JPY, intPrice(1), priceNow)
	env = currentRequest(t, s, "/products/p/prices", 200)
	if !strings.Contains(string(env["data"]), `"best_price":null`) || !strings.Contains(string(env["data"]), `"comparison_status":"incompatible_currencies"`) {
		t.Fatal("mixed currencies produced a best price")
	}
	currentRequest(t, currentStub{}, "/products/p/prices", 200)
	for _, test := range []struct {
		path string
		s    currentStub
		code int
	}{
		{"/listings/l/price", currentStub{err: sql.ErrNoRows}, 404}, {"/listings/l/price", currentStub{err: errors.New("secret")}, 500},
		{"/products/p/prices", currentStub{productErr: sql.ErrNoRows}, 404}, {"/products/p/prices", currentStub{productErr: errors.New("secret")}, 500},
		{"/products/p/prices", currentStub{err: errors.New("secret")}, 500}, {"/products/p/prices", currentStub{err: persistence.ErrTooManyListings}, 422},
	} {
		currentRequest(t, test.s, test.path, test.code)
	}
}
