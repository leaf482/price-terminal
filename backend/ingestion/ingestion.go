// Package ingestion collects and persists listings through single-listing attempts
// and bounded sequential batches. Providers never receive the store. Scheduling,
// automatic retries, and persistent failure recording remain outside this package;
// errors are returned without creating failure observations.
package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/provider"
)

// Store is the existing persistence functionality needed for a single write.
type Store interface {
	GetListing(context.Context, string) (domain.Listing, error)
	InsertPriceObservation(context.Context, string, domain.PriceObservation) error
}

type Ingestor struct {
	provider provider.Provider
	store    Store
}

func New(p provider.Provider, store Store) *Ingestor {
	return &Ingestor{provider: p, store: store}
}

// Collected retains an immutable snapshot and its caller-supplied persistence ID.
// Only a successful, validated collection creates this value. Its zero value is
// invalid. Retain it if persistence fails; retry Persist, not Ingest.
type Collected struct {
	id           string
	listing      domain.Listing
	observation  domain.PriceObservation
	fromProvider bool
}

func (c Collected) ID() string                           { return c.id }
func (c Collected) Observation() domain.PriceObservation { return c.observation }

// Ingest calls the provider exactly once after input validation, then attempts
// one atomic observation insert. resultID must be globally unique per independent
// collection. Neither timestamps nor source facts are generated or refreshed here.
// Collection/validation failures return a zero Collected; persistence failures
// return the validated Collected plus the error, allowing an exact write retry.
// Callers supply the context/deadline; no background work or retry loop is started.
func (i *Ingestor) Ingest(ctx context.Context, resultID string, listing domain.Listing) (Collected, error) {
	if err := ctx.Err(); err != nil {
		return Collected{}, fmt.Errorf("ingest: %w", err)
	}
	if strings.TrimSpace(resultID) == "" {
		return Collected{}, fmt.Errorf("ingest: result ID is required")
	}
	if err := listing.Validate(); err != nil {
		return Collected{}, fmt.Errorf("ingest listing: %w", err)
	}
	if listing.TrackingDisabled {
		return Collected{}, domain.ErrTrackingDisabled
	}
	// Scheduled jobs may wait behind another Listing; recheck before contacting
	// the provider rather than relying on the cycle's earlier snapshot.
	current, err := i.store.GetListing(ctx, listing.ID)
	if err != nil {
		return Collected{}, fmt.Errorf("ingest listing lookup: %w", err)
	}
	if current.TrackingDisabled {
		return Collected{}, domain.ErrTrackingDisabled
	}
	observation, err := i.provider.Collect(ctx, listing)
	if err != nil {
		slog.Warn("ingestion_failure", "listing_id", listing.ID, "observation_id", resultID, "stage", "provider")
		return Collected{}, fmt.Errorf("ingest collect: %w", err)
	}
	if err := provider.ValidateResult(listing, observation); err != nil {
		slog.Warn("ingestion_failure", "listing_id", listing.ID, "observation_id", resultID, "stage", "validation")
		return Collected{}, fmt.Errorf("ingest result: %w", err)
	}
	collected := Collected{id: resultID, listing: listing, observation: observation, fromProvider: true}
	return collected, i.Persist(ctx, collected)
}

// Record accepts already observed facts, including manual entries, without a
// provider call. It uses the same validation, persistence and alert evaluation.
func (i *Ingestor) Record(ctx context.Context, resultID string, listing domain.Listing, observation domain.PriceObservation) (Collected, error) {
	collected := Collected{id: resultID, listing: listing, observation: observation}
	return collected, i.Persist(ctx, collected)
}

// Persist writes a previously validated collection without calling the provider.
// A repeated ID remains a wrapped uniqueness error, including when a previous
// write succeeded but its acknowledgement was lost. Never overwrite or silently
// treat conflicting IDs as success. The original ID, facts, and time are reused.
func (i *Ingestor) Persist(ctx context.Context, collected Collected) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("ingest persist: %w", err)
	}
	if strings.TrimSpace(collected.id) == "" {
		return fmt.Errorf("ingest persist: result ID is required")
	}
	if err := provider.ValidateResult(collected.listing, collected.observation); err != nil {
		return fmt.Errorf("ingest persist result: %w", err)
	}
	stored, err := i.store.GetListing(ctx, collected.listing.ID)
	if err != nil {
		return fmt.Errorf("ingest listing lookup: %w", err)
	}
	// Reject stale or fabricated source/relationship context instead of attaching
	// facts collected for it to a different stored listing. No related rows are created.
	if collected.fromProvider && stored.TrackingDisabled {
		return domain.ErrTrackingDisabled
	}
	// Tracking is mutable operational state, not source identity.
	expected := collected.listing
	expected.TrackingDisabled = stored.TrackingDisabled
	if stored != expected {
		return fmt.Errorf("ingest persist: listing differs from stored source or relationships")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("ingest persist: %w", err)
	}
	write := i.store.InsertPriceObservation
	if collected.fromProvider {
		if guarded, ok := i.store.(interface {
			InsertCollectedObservation(context.Context, string, domain.PriceObservation) error
		}); ok {
			write = guarded.InsertCollectedObservation
		}
	}
	if err := write(ctx, collected.id, collected.observation); err != nil {
		slog.Warn("ingestion_failure", "listing_id", collected.listing.ID, "observation_id", collected.id, "stage", "persistence")
		return fmt.Errorf("ingest persist: %w", err)
	}
	// Alert evaluation is downstream of the committed observation. Its failure
	// must not turn a successful write into a retryable ingestion failure.
	if evaluator, ok := i.store.(interface {
		EvaluateAlerts(context.Context, string) error
	}); ok {
		if err := evaluator.EvaluateAlerts(ctx, collected.id); err != nil {
			slog.Warn("alert_evaluation_failure", "listing_id", collected.listing.ID, "observation_id", collected.id)
		}
	}
	return nil
}
