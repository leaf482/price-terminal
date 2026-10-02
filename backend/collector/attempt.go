package collector

import (
	"context"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"log/slog"
	"time"
)

// Recording is outside the observation transaction. Cancellation must still be
// auditable, but metadata may delay shutdown by at most two seconds per attempt.
func (r *Runtime) record(ctx context.Context, trigger string, o ingestion.Outcome) {
	if !o.Started {
		return
	}
	writer, ok := r.store.(interface {
		InsertCollectionAttempt(context.Context, domain.CollectionAttempt) error
	})
	if !ok {
		return
	}
	a := domain.CollectionAttempt{ID: o.ResultID, ListingID: o.Listing.ID, Trigger: trigger, StartedAt: o.StartedAt, FinishedAt: o.FinishedAt, Outcome: "success"}
	switch {
	case o.Err == nil:
		a.ObservationID = o.Collected.ID()
	case errors.Is(o.Err, context.Canceled) || errors.Is(o.Err, context.DeadlineExceeded):
		a.Outcome = "cancelled"
	case errors.Is(o.Err, domain.ErrTrackingDisabled) || errors.Is(o.Err, ErrUnavailable):
		a.Outcome = "unavailable"
	case o.Collected.ID() != "" || errors.Is(o.Err, ingestion.ErrListingLookup):
		a.Outcome = "persistence_error"
	default:
		a.Outcome = "provider_error"
	}
	if o.Err != nil {
		a.ErrorSummary = a.Outcome
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if err := writer.InsertCollectionAttempt(auditCtx, a); err != nil {
		slog.Warn("collection_attempt_record_failed", "listing_id", a.ListingID, "attempt_id", a.ID)
	}
}
