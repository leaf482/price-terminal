package provider_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/provider"
)

func TestFakeSuccessfulCollection(t *testing.T) {
	for _, currency := range []domain.Currency{domain.USD, domain.JPY} {
		for _, mode := range []string{"full", "stock only", "missing prices", "zero"} {
			t.Run(string(currency)+"/"+mode, func(t *testing.T) {
				input := facts()
				switch mode {
				case "full":
					input.MSRP = &domain.Money{MinorUnits: 12000, Currency: currency}
					input.MSRPSource = "fixture manufacturer claim"
					input.RetailerListPrice = &domain.Money{MinorUnits: 11000, Currency: currency}
					input.SalePrice = &domain.Money{MinorUnits: 9000, Currency: currency}
					input.OfferPrice = &domain.Money{MinorUnits: 9500, Currency: currency}
					input.Stock = domain.StockInStock
				case "stock only":
					input.Stock = domain.StockOutOfStock
				case "missing prices":
					input.OfferPrice = &domain.Money{MinorUnits: 500, Currency: currency}
				case "zero":
					input.OfferPrice = &domain.Money{Currency: currency}
				}
				want, err := domain.NewPriceObservation(input)
				if err != nil {
					t.Fatal(err)
				}
				config := map[string]provider.FakeResponse{listing().ID: {Observation: want}}
				fake := provider.NewFake(config)
				// Neither caller map edits nor constructor input pointer edits can
				// alter the fake's stored snapshot.
				config[listing().ID] = provider.FakeResponse{Err: errors.New("replacement")}
				if input.OfferPrice != nil {
					*input.OfferPrice = domain.Money{}
				}
				request := listing()
				for range 3 {
					got, err := fake.Collect(context.Background(), request)
					if err != nil || !reflect.DeepEqual(got, want) {
						t.Fatalf("got %+v, %v; want %+v", got, err, want)
					}
					if err := provider.ValidateResult(request, got); err != nil {
						t.Fatal(err)
					}
					if mode == "stock only" {
						if c, present := got.Currency(); present || c != "" {
							t.Fatal("stock-only result invented currency")
						}
					}
					price, _ := got.OfferPrice()
					price.MinorUnits = -1
				}
				if request != listing() {
					t.Fatal("request identity changed")
				}
			})
		}
	}
}

func TestFakeFailuresAndIdentity(t *testing.T) {
	want, err := domain.NewPriceObservation(facts())
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("configured collection failure")
	fake := provider.NewFake(map[string]provider.FakeResponse{
		"l1":       {Observation: want},
		"failed":   {Observation: want, Err: cause},
		"invalid":  {},
		"mismatch": {Observation: want},
	})
	for _, id := range []string{"failed", "invalid", "mismatch", "unknown", ""} {
		t.Run(id, func(t *testing.T) {
			request := listing()
			request.ID = id
			got, err := fake.Collect(context.Background(), request)
			if err == nil || !reflect.DeepEqual(got, domain.PriceObservation{}) {
				t.Fatalf("expected failure with no observation, got %+v, %v", got, err)
			}
			if id == "failed" && !errors.Is(err, cause) {
				t.Fatalf("configured cause lost: %v", err)
			}
		})
	}
	// A different configured listing returns its own facts, not another fixture.
	input := facts()
	input.ListingID = "l2"
	second, err := domain.NewPriceObservation(input)
	if err != nil {
		t.Fatal(err)
	}
	fake = provider.NewFake(map[string]provider.FakeResponse{"l1": {Observation: want}, "l2": {Observation: second}})
	request := listing()
	request.ID = "l2"
	got, err := fake.Collect(context.Background(), request)
	if err != nil || !reflect.DeepEqual(got, second) {
		t.Fatalf("wrong per-listing response: %+v, %v", got, err)
	}
	request.URL = "/relative"
	if _, err := fake.Collect(context.Background(), request); err == nil {
		t.Fatal("invalid listing accepted")
	}
	var empty provider.Fake
	if _, err := empty.Collect(context.Background(), listing()); err == nil {
		t.Fatal("unconfigured zero-value fake accepted listing")
	}
}

func TestFakeCanceledContext(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, cancelDeadline := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancelDeadline()
	for _, ctx := range []context.Context{canceled, expired} {
		got, err := provider.NewFake(nil).Collect(ctx, listing())
		if !errors.Is(err, ctx.Err()) || !reflect.DeepEqual(got, domain.PriceObservation{}) {
			t.Fatalf("expected %v with no observation, got %+v, %v", ctx.Err(), got, err)
		}
	}
}
