package domain

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func TestEffectivePrice(t *testing.T) {
	now := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	yes := PromotionScenario{Member: "yes", Eligible: "yes"}
	fixed := Promotion{ID: "p", ListingID: "l", Source: "fixture", ObservedAt: now, Kind: "fixed", Amount: &Money{100, USD}, Requirement: "none", Stacking: "allowed", Terms: "instant discount"}
	for _, tc := range []struct {
		name           string
		base           int64
		edit           func(*Promotion)
		extra          bool
		scenario       PromotionScenario
		status, reason string
		pay, net       int64
	}{
		{name: "fixed", base: 1000, status: "available", pay: 900, net: 900},
		{name: "zero", base: 100, status: "available", pay: 0, net: 0},
		{name: "round half up", base: 101, edit: func(p *Promotion) { p.Kind = "percentage"; p.Amount = nil; n := int64(5000); p.BasisPoints = &n }, status: "available", pay: 50, net: 50},
		{name: "round below half", base: 104, edit: func(p *Promotion) { p.Kind = "percentage"; p.Amount = nil; n := int64(1000); p.BasisPoints = &n }, status: "available", pay: 94, net: 94},
		{name: "overflow safe", base: math.MaxInt64, edit: func(p *Promotion) { p.Kind = "percentage"; p.Amount = nil; n := int64(10000); p.BasisPoints = &n }, status: "available", pay: 0, net: 0},
		{name: "cashback", base: 1000, edit: func(p *Promotion) { p.Kind = "cashback" }, status: "available", pay: 1000, net: 900},
		{name: "stacked", base: 1000, extra: true, status: "available", pay: 900, net: 850},
		{name: "membership", base: 1000, edit: func(p *Promotion) { p.Kind = "membership"; p.Requirement = "membership" }, status: "available", pay: 900, net: 900},
		{name: "unknown member", base: 1000, edit: func(p *Promotion) { p.Kind = "membership"; p.Requirement = "membership" }, scenario: PromotionScenario{"unknown", "yes"}, status: "conditional", reason: "unknown_membership"},
		{name: "not member", base: 1000, edit: func(p *Promotion) { p.Kind = "membership"; p.Requirement = "membership" }, scenario: PromotionScenario{"no", "yes"}, status: "unavailable", reason: "membership_required"},
		{name: "unknown eligibility", base: 1000, edit: func(p *Promotion) { p.Requirement = "unknown" }, status: "conditional", reason: "unknown_eligibility"},
		{name: "other unconfirmed", base: 1000, edit: func(p *Promotion) { p.Requirement = "other" }, scenario: PromotionScenario{"unknown", "unknown"}, status: "conditional", reason: "unknown_eligibility"},
		{name: "unknown stacking", base: 1000, extra: true, edit: func(p *Promotion) { p.Stacking = "unknown" }, status: "conditional", reason: "unknown_stacking"},
		{name: "disallowed stacking", base: 1000, extra: true, edit: func(p *Promotion) { p.Stacking = "disallowed" }, status: "unavailable", reason: "stacking_disallowed"},
		{name: "currency", base: 1000, edit: func(p *Promotion) { p.Amount = &Money{100, JPY} }, status: "unavailable", reason: "incompatible_currency"},
		{name: "expired inclusive end", base: 1000, edit: func(p *Promotion) { v := now; p.EndsAt = &v }, status: "unavailable", reason: "expired"},
		{name: "upcoming", base: 1000, edit: func(p *Promotion) { v := now.Add(time.Second); p.StartsAt = &v }, status: "unavailable", reason: "not_yet_active"},
		{name: "future evidence", base: 1000, edit: func(p *Promotion) { p.ObservedAt = now.Add(time.Second) }, status: "unavailable", reason: "not_yet_active"},
		{name: "wrong listing", base: 1000, edit: func(p *Promotion) { p.ListingID = "other" }, status: "unavailable", reason: "listing_mismatch"},
		{name: "too large", base: 99, status: "unavailable", reason: "discount_exceeds_base"},
		{name: "rebate too large", base: 99, edit: func(p *Promotion) { p.Kind = "cashback" }, status: "unavailable", reason: "cashback_exceeds_payable"},
		{name: "two rebates unsupported", base: 1000, extra: true, edit: func(p *Promotion) { p.Kind = "cashback" }, status: "unavailable", reason: "unsupported_combination"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := fixed
			if tc.edit != nil {
				tc.edit(&p)
			}
			promotions := []Promotion{p}
			if tc.extra {
				q := fixed
				q.ID = "cashback"
				q.Kind = "cashback"
				q.Amount = &Money{50, USD}
				promotions = append(promotions, q)
			}
			s := tc.scenario
			if s.Member == "" {
				s = yes
			}
			o, err := NewPriceObservation(PriceObservationInput{ListingID: "l", ObservedAt: now, Source: "price", Stock: StockInStock, OfferPrice: &Money{tc.base, USD}, SalePrice: &Money{1, USD}})
			if err != nil {
				t.Fatal(err)
			}
			before := o
			r, err := CalculateEffectivePrice(o, promotions, s, now)
			if err != nil || r.Status != tc.status {
				t.Fatal(r, err)
			}
			if tc.reason != "" && r.Reason != tc.reason {
				t.Fatal(r.Reason)
			}
			if tc.status == "available" {
				if r.Payable.MinorUnits != tc.pay || r.Net.MinorUnits != tc.net || r.Cashback.MinorUnits != tc.pay-tc.net {
					t.Fatal(r)
				}
			} else if r.Payable != nil || r.Net != nil {
				t.Fatal("conditional total leaked")
			}
			if !reflect.DeepEqual(o, before) {
				t.Fatal("observation mutated")
			}
		})
	}
	t.Run("duplicate selection", func(t *testing.T) {
		o, _ := NewPriceObservation(PriceObservationInput{ListingID: "l", ObservedAt: now, Source: "x", Stock: StockUnknown, SalePrice: &Money{1000, USD}})
		r, err := CalculateEffectivePrice(o, []Promotion{fixed, fixed}, yes, now)
		if err != nil || r.Reason != "duplicate_promotion" {
			t.Fatal(r, err)
		}
	})
}
func TestPromotionValidation(t *testing.T) {
	valid := Promotion{ID: "p", ListingID: "l", Source: "s", ObservedAt: time.Now(), Kind: "fixed", Amount: &Money{0, USD}, Requirement: "unknown", Stacking: "unknown", Terms: "terms"}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, edit := range []func(*Promotion){func(p *Promotion) { p.ID = "" }, func(p *Promotion) { p.ListingID = "" }, func(p *Promotion) { p.Source = "" }, func(p *Promotion) { p.Terms = "" }, func(p *Promotion) { p.ObservedAt = time.Time{} }, func(p *Promotion) { p.Amount = &Money{-1, USD} }, func(p *Promotion) { p.Amount = &Money{1, "EUR"} }, func(p *Promotion) { p.Kind = "percentage" }, func(p *Promotion) { p.Kind = "membership" }, func(p *Promotion) { p.Kind = "unknown" }, func(p *Promotion) { p.Requirement = "" }, func(p *Promotion) { p.Stacking = "" }, func(p *Promotion) { v := p.ObservedAt; p.StartsAt = &v; p.EndsAt = &v }} {
		p := valid
		edit(&p)
		if p.Validate() == nil {
			t.Fatal("invalid promotion accepted", p)
		}
	}
}
