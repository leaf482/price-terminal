package domain

import (
	"fmt"
	"math/big"
	"time"
)

// Promotion is one evidence snapshot. Persistence only appends; a new claim
// receives a new ID, never an update of an earlier snapshot.
type Promotion struct {
	ID          string
	ListingID   string
	Source      string
	ObservedAt  time.Time
	StartsAt    *time.Time
	EndsAt      *time.Time
	Kind        string // fixed, percentage, cashback, membership (fixed amount)
	Amount      *Money
	BasisPoints *int64 // 100 basis points = 1%; percentage only
	Requirement string // none, membership, other, unknown
	Stacking    string // allowed, disallowed, unknown
	Terms       string // coupon code/activation and source conditions, preserved verbatim
}

func (p Promotion) Validate() error {
	for _, v := range []string{p.ID, p.ListingID, p.Source, p.Terms} {
		if err := requireText("promotion identity/source/terms", v); err != nil {
			return err
		}
	}
	if p.ObservedAt.IsZero() {
		return fmt.Errorf("promotion observation time required")
	}
	for _, v := range []*time.Time{p.StartsAt, p.EndsAt} {
		if v != nil && v.IsZero() {
			return fmt.Errorf("invalid validity time")
		}
	}
	if p.StartsAt != nil && p.EndsAt != nil && !p.EndsAt.After(*p.StartsAt) {
		return fmt.Errorf("end must follow start")
	}
	switch p.Kind {
	case "percentage":
		if p.Amount != nil || p.BasisPoints == nil || *p.BasisPoints < 1 || *p.BasisPoints > 10000 {
			return fmt.Errorf("percentage requires 1..10000 basis points only")
		}
	case "fixed", "cashback", "membership":
		if p.Amount == nil || p.BasisPoints != nil {
			return fmt.Errorf("monetary promotion requires amount only")
		}
		if err := p.Amount.Validate(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported promotion kind")
	}
	switch p.Requirement {
	case "none", "membership", "other", "unknown":
	default:
		return fmt.Errorf("invalid requirement")
	}
	if p.Kind == "membership" && p.Requirement != "membership" {
		return fmt.Errorf("membership discount must require membership")
	}
	switch p.Stacking {
	case "allowed", "disallowed", "unknown":
	default:
		return fmt.Errorf("invalid stacking")
	}
	return nil
}

type PromotionScenario struct {
	Member   string `json:"member"`   // yes, no, unknown; caller assumption, not profile data
	Eligible string `json:"eligible"` // confirms all other documented conditions
}

func (s PromotionScenario) Validate() error {
	for _, v := range []string{s.Member, s.Eligible} {
		if v != "yes" && v != "no" && v != "unknown" {
			return fmt.Errorf("scenario values must be yes, no, or unknown")
		}
	}
	return nil
}

type EffectivePrice struct {
	Status            string
	Reason            string
	Basis             string
	Base              *Money
	ImmediateDiscount *Money
	Payable           *Money
	Cashback          *Money
	Net               *Money
}

// CalculateEffectivePrice supports one immediate discount, one fixed cashback,
// or both with explicit stacking permission. Percentage discount rounding is
// nearest minor unit, halves up, using big integers to avoid multiplication overflow.
// Unknown conditions never produce numeric derived totals. Inputs remain untouched.
func CalculateEffectivePrice(o PriceObservation, promotions []Promotion, scenario PromotionScenario, at time.Time) (EffectivePrice, error) {
	result := EffectivePrice{Status: "unavailable"}
	if err := o.Validate(); err != nil {
		return result, err
	}
	if err := scenario.Validate(); err != nil {
		return result, err
	}
	if at.IsZero() {
		return result, fmt.Errorf("calculation time required")
	}
	base, ok := o.OfferPrice()
	result.Basis = "offer_price"
	if !ok {
		base, ok = o.SalePrice()
		result.Basis = "sale_price"
	}
	if !ok {
		result.Reason = "missing_observed_price"
		return result, nil
	}
	result.Base = &base
	if o.ObservedAt().After(at) {
		result.Reason = "future_observation"
		return result, nil
	}
	if len(promotions) < 1 || len(promotions) > 2 {
		result.Reason = "select_one_or_two_promotions"
		return result, nil
	}
	immediate, cashback := 0, 0
	ids := map[string]bool{}
	conditional := ""
	for _, p := range promotions {
		if err := p.Validate(); err != nil {
			return result, err
		}
		if p.ListingID != o.ListingID() {
			result.Reason = "listing_mismatch"
			return result, nil
		}
		if ids[p.ID] {
			result.Reason = "duplicate_promotion"
			return result, nil
		}
		ids[p.ID] = true
		if p.Kind == "cashback" {
			cashback++
		} else {
			immediate++
		}
		if p.Amount != nil && p.Amount.Currency != base.Currency {
			result.Reason = "incompatible_currency"
			return result, nil
		}
		if p.ObservedAt.After(at) || (p.StartsAt != nil && at.Before(*p.StartsAt)) {
			result.Reason = "not_yet_active"
			return result, nil
		}
		if p.EndsAt != nil && !at.Before(*p.EndsAt) {
			result.Reason = "expired"
			return result, nil
		}
		if p.Requirement == "unknown" {
			conditional = "unknown_eligibility"
		}
		if p.Requirement == "membership" {
			if scenario.Member == "no" {
				result.Reason = "membership_required"
				return result, nil
			}
			if scenario.Member == "unknown" {
				conditional = "unknown_membership"
			}
		}
		if p.Requirement != "none" {
			if scenario.Eligible == "no" {
				result.Reason = "not_eligible"
				return result, nil
			}
			if scenario.Eligible == "unknown" {
				conditional = "unknown_eligibility"
			}
		}
		if len(promotions) > 1 {
			if p.Stacking == "disallowed" {
				result.Reason = "stacking_disallowed"
				return result, nil
			}
			if p.Stacking == "unknown" {
				conditional = "unknown_stacking"
			}
		}
	}
	if immediate > 1 || cashback > 1 {
		result.Reason = "unsupported_combination"
		return result, nil
	}
	if conditional != "" {
		result.Status = "conditional"
		result.Reason = conditional
		return result, nil
	}
	var discount, rebate int64
	for _, p := range promotions {
		if p.Kind == "cashback" {
			rebate = p.Amount.MinorUnits
			continue
		}
		if p.Kind == "percentage" {
			n := new(big.Int).Mul(big.NewInt(base.MinorUnits), big.NewInt(*p.BasisPoints))
			n.Add(n, big.NewInt(5000))
			n.Quo(n, big.NewInt(10000))
			discount = n.Int64()
		} else {
			discount = p.Amount.MinorUnits
		}
	}
	if discount > base.MinorUnits {
		result.Reason = "discount_exceeds_base"
		return result, nil
	}
	payable := base.MinorUnits - discount
	if rebate > payable {
		result.Reason = "cashback_exceeds_payable"
		return result, nil
	}
	result.Status = "available"
	result.Reason = "supported_scenario"
	result.ImmediateDiscount = &Money{MinorUnits: discount, Currency: base.Currency}
	result.Payable = &Money{MinorUnits: payable, Currency: base.Currency}
	result.Cashback = &Money{MinorUnits: rebate, Currency: base.Currency}
	result.Net = &Money{MinorUnits: payable - rebate, Currency: base.Currency}
	return result, nil
}
