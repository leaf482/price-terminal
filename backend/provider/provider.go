// Package provider defines the single-listing collection boundary. Providers
// normalize source claims; they do not access persistence, calculate effective
// prices, schedule work, or assign persistence retry identities.
package provider

import (
	"context"
	"fmt"

	"github.com/leaf482/price-terminal/backend/domain"
)

// Provider collects one listing's observed facts. Listing supplies its identity,
// retailer context, exact source URL, and optional retailer product/variant ID.
// Implementations must validate the listing and must not normalize its identity
// or look it up in a database.
//
// A successful call returns an observation constructed with
// domain.NewPriceObservation and a nil error. Missing prices and unknown stock
// are valid source outcomes, including stock-only observations without currency.
// Source, observation time, and explicit MSRP evidence must come from collection;
// they must not be invented to make a malformed result pass validation.
// ValidateResult checks the normalized observation and requested listing identity.
//
// A failed collection returns a non-nil error; callers must ignore its result.
// Fetch/parse/access failures must never become empty successful observations.
// Error categories beyond ordinary Go errors are deferred until actual adapters
// need them. Wrapped causes must remain available through errors.Is/errors.As.
//
// Implementations must honor ctx cancellation/deadlines during all blocking work,
// return promptly for an already-canceled context, and preserve context.Canceled
// or context.DeadlineExceeded in the error chain. An interface cannot enforce
// cancellation: each implementation must test it. Callers supply a non-nil context
// and choose deadlines. No default timeout or retry policy is imposed here.
type Provider interface {
	Collect(ctx context.Context, listing domain.Listing) (domain.PriceObservation, error)
}

// ValidateResult checks a successful collection only. Callers must first check
// Collect's error; even a valid-looking observation must be ignored on failure.
// Providers and future ingestion can share this check without database access.
func ValidateResult(listing domain.Listing, observation domain.PriceObservation) error {
	if err := listing.Validate(); err != nil {
		return fmt.Errorf("provider listing: %w", err)
	}
	if err := observation.Validate(); err != nil {
		return fmt.Errorf("provider result: %w", err)
	}
	if observation.ListingID() != listing.ID {
		return fmt.Errorf("provider result: listing ID does not match requested listing")
	}
	return nil
}
