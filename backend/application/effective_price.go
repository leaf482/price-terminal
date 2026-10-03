// Package application coordinates use cases without owning HTTP or transactions.
// Existing ingestion and collector packages remain the owners of write workflows.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

// EffectivePriceStore contains only the reads required by this use case.
type EffectivePriceStore interface {
	CurrentListing(context.Context, string) (persistence.CurrentListing, error)
	GetPromotion(context.Context, string, string) (domain.Promotion, error)
}

// Stage markers let callers map errors without losing the original store cause.
var (
	ErrCurrentListing    = errors.New("read current listing")
	ErrSelectedPromotion = errors.New("read selected promotion")
	ErrEffectivePrice    = errors.New("calculate effective price")
)

type EffectivePriceResult struct {
	Observation *domain.PriceObservation
	Promotions  []domain.Promotion
	Price       domain.EffectivePrice
}

// ReadEffectivePrice loads evidence in selection order before calculating a
// scenario. The caller supplies validated selection/eligibility and one clock
// instant. Reads retain their existing independent-query semantics: no new
// transaction, retry, write, or alert evaluation is introduced.
func ReadEffectivePrice(ctx context.Context, store EffectivePriceStore, listingID string, promotionIDs []string, scenario domain.PromotionScenario, now time.Time) (EffectivePriceResult, error) {
	current, err := store.CurrentListing(ctx, listingID)
	if err != nil {
		return EffectivePriceResult{}, fmt.Errorf("%w: %w", ErrCurrentListing, err)
	}
	selected := make([]domain.Promotion, 0, len(promotionIDs))
	for _, id := range promotionIDs {
		p, err := store.GetPromotion(ctx, listingID, id)
		if err != nil {
			return EffectivePriceResult{}, fmt.Errorf("%w: %w", ErrSelectedPromotion, err)
		}
		selected = append(selected, p)
	}
	result := EffectivePriceResult{Observation: current.Observation, Promotions: selected, Price: domain.EffectivePrice{Status: "unavailable", Reason: "missing_observation"}}
	if current.Observation != nil {
		result.Price, err = domain.CalculateEffectivePrice(*current.Observation, selected, scenario, now)
		if err != nil {
			return EffectivePriceResult{}, fmt.Errorf("%w: %w", ErrEffectivePrice, err)
		}
	}
	return result, nil
}
