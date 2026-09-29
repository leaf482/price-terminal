package provider

import (
	"context"
	"fmt"

	"github.com/leaf482/price-terminal/backend/domain"
)

// FakeResponse configures one listing's fixed outcome. Observation should be
// built with domain.NewPriceObservation, including an explicit fixture time.
// When Err is non-nil, it takes precedence and no observation is returned.
type FakeResponse struct {
	Observation domain.PriceObservation
	Err         error
}

// Fake returns fixed outcomes without I/O, delays, or changes to source facts.
// Its zero value has no configured listings.
type Fake struct {
	responses map[string]FakeResponse
}

var _ Provider = (*Fake)(nil)

// NewFake copies responses keyed by exact Listing ID. Observations are immutable
// domain values. Configured errors are retained as supplied, preserving their
// identity and type; callers must not mutate custom error values after setup.
func NewFake(responses map[string]FakeResponse) *Fake {
	f := &Fake{responses: make(map[string]FakeResponse, len(responses))}
	for id, response := range responses {
		f.responses[id] = response
	}
	return f
}

// Collect validates the request and successful fixture on every call. Unknown
// listings and invalid fixtures fail rather than becoming empty observations.
// Repeated calls return the same configured facts, including observation time.
func (f *Fake) Collect(ctx context.Context, listing domain.Listing) (domain.PriceObservation, error) {
	if err := ctx.Err(); err != nil {
		return domain.PriceObservation{}, fmt.Errorf("fake collect: %w", err)
	}
	if err := listing.Validate(); err != nil {
		return domain.PriceObservation{}, fmt.Errorf("fake collect: %w", err)
	}
	response, ok := f.responses[listing.ID]
	if !ok {
		return domain.PriceObservation{}, fmt.Errorf("fake collect: listing %q is not configured", listing.ID)
	}
	if response.Err != nil {
		return domain.PriceObservation{}, fmt.Errorf("fake collect: %w", response.Err)
	}
	if err := ValidateResult(listing, response.Observation); err != nil {
		return domain.PriceObservation{}, fmt.Errorf("fake collect: %w", err)
	}
	return response.Observation, nil
}
