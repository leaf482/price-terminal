package application

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

type evidenceStore struct {
	current   func(context.Context, string) (persistence.CurrentListing, error)
	promotion func(context.Context, string, string) (domain.Promotion, error)
}

func (s evidenceStore) CurrentListing(c context.Context, id string) (persistence.CurrentListing, error) {
	return s.current(c, id)
}
func (s evidenceStore) GetPromotion(c context.Context, l, id string) (domain.Promotion, error) {
	return s.promotion(c, l, id)
}

func TestReadEffectivePrice(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 123, time.UTC)
	scenario := domain.PromotionScenario{Member: "unknown", Eligible: "unknown"}
	for _, tt := range []struct {
		name    string
		amount  *domain.Money
		missing bool
	}{
		{"priced", &domain.Money{MinorUnits: 1000, Currency: domain.USD}, false},
		{"explicit zero", &domain.Money{MinorUnits: 0, Currency: domain.USD}, false},
		{"stock only", nil, false},
		{"no observation", nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: now, Source: "fixture", Stock: domain.StockInStock, OfferPrice: tt.amount})
			if err != nil {
				t.Fatal(err)
			}
			var observation *domain.PriceObservation
			if !tt.missing {
				observation = &o
			}
			promotions := []domain.Promotion{
				{ID: "second", ListingID: "l", Source: "fixture", ObservedAt: now, Kind: "fixed", Amount: &domain.Money{MinorUnits: 0, Currency: domain.USD}, Requirement: "none", Stacking: "allowed", Terms: "explicit zero discount"},
				{ID: "first", ListingID: "l", Source: "fixture", ObservedAt: now, Kind: "cashback", Amount: &domain.Money{MinorUnits: 0, Currency: domain.USD}, Requirement: "none", Stacking: "allowed", Terms: "explicit zero cashback"},
			}
			calls := []string{}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			store := evidenceStore{
				current: func(c context.Context, id string) (persistence.CurrentListing, error) {
					if c != ctx || id != "l" {
						t.Fatal("context or identity changed")
					}
					calls = append(calls, "current")
					return persistence.CurrentListing{Observation: observation}, nil
				},
				promotion: func(c context.Context, l, id string) (domain.Promotion, error) {
					if c != ctx || l != "l" {
						t.Fatal("context or listing changed")
					}
					calls = append(calls, id)
					if id == "second" {
						return promotions[0], nil
					}
					return promotions[1], nil
				},
			}
			got, err := ReadEffectivePrice(ctx, store, "l", []string{"second", "first"}, scenario, now)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []string{"current", "second", "first"}) || !reflect.DeepEqual(got.Promotions, promotions) || got.Observation != observation {
				t.Fatal("evidence order/facts changed", calls, got)
			}
			want := domain.EffectivePrice{Status: "unavailable", Reason: "missing_observation"}
			if observation != nil {
				want, err = domain.CalculateEffectivePrice(o, promotions, scenario, now)
				if err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(got.Price, want) {
				t.Fatal("domain result changed", got.Price, want)
			}
		})
	}
}

func TestEffectivePriceReadFailures(t *testing.T) {
	for _, stage := range []string{"listing", "promotion"} {
		for _, cause := range []error{sql.ErrNoRows, errors.New("private database detail"), context.Canceled, context.DeadlineExceeded} {
			t.Run(stage+"/"+cause.Error(), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				if errors.Is(cause, context.Canceled) {
					cancel()
				}
				defer cancel()
				calls := 0
				store := evidenceStore{
					current: func(c context.Context, _ string) (persistence.CurrentListing, error) {
						if c != ctx {
							t.Fatal("context replaced")
						}
						if stage == "listing" {
							return persistence.CurrentListing{}, cause
						}
						return persistence.CurrentListing{}, nil
					},
					promotion: func(c context.Context, _, _ string) (domain.Promotion, error) {
						calls++
						if c != ctx {
							t.Fatal("context replaced")
						}
						return domain.Promotion{}, cause
					},
				}
				_, err := ReadEffectivePrice(ctx, store, "l", []string{"p", "unread"}, domain.PromotionScenario{}, time.Time{})
				marker := ErrCurrentListing
				if stage == "promotion" {
					marker = ErrSelectedPromotion
				}
				if !errors.Is(err, cause) || !errors.Is(err, marker) {
					t.Fatal("error identity lost", err)
				}
				if stage == "listing" && calls != 0 || stage == "promotion" && calls != 1 {
					t.Fatal("work continued after failure", calls)
				}
			})
		}
	}
}

func TestEffectivePriceCalculationFailure(t *testing.T) {
	// Invalid stored facts remain a calculation failure; they are not hidden as
	// a successful missing-observation response.
	invalid := domain.PriceObservation{}
	store := evidenceStore{current: func(context.Context, string) (persistence.CurrentListing, error) {
		return persistence.CurrentListing{Observation: &invalid}, nil
	}}
	_, err := ReadEffectivePrice(context.Background(), store, "l", nil, domain.PromotionScenario{Member: "unknown", Eligible: "unknown"}, time.Now())
	if !errors.Is(err, ErrEffectivePrice) {
		t.Fatal(err)
	}
}
