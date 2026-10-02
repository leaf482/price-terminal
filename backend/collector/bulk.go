package collector

import (
	"context"
	"database/sql"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"strings"
)

const MaxBulkListings = 20

var ErrInvalidBatch = errors.New("provide 1..20 nonblank Listing IDs")

type BulkResult struct {
	ListingID     string `json:"listing_id"`
	Outcome       string `json:"outcome"`
	ErrorSummary  string `json:"error_summary,omitempty"`
	ObservationID string `json:"observation_id,omitempty"`
}

// Validate before any work. Deduplication retains the first occurrence and never
// rewrites IDs. The raw input is bounded as well as the unique work set.
func BulkIDs(ids []string) ([]string, error) {
	if len(ids) == 0 || len(ids) > MaxBulkListings {
		return nil, ErrInvalidBatch
	}
	unique := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			return nil, ErrInvalidBatch
		}
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	return unique, nil
}

// Sequential manual attempts use the exact single-Listing path, including its
// guard, deadline, observation write and best-effort attempt audit. Cancellation
// prevents all subsequent calls; already committed observations remain successes.
func (r *Runtime) CollectBulk(ctx context.Context, ids []string) ([]BulkResult, error) {
	ids, err := BulkIDs(ids)
	if err != nil {
		return nil, err
	}
	results := make([]BulkResult, 0, len(ids))
	for _, id := range ids {
		v := BulkResult{ListingID: id, Outcome: "success"}
		var err error
		if ctx.Err() != nil {
			err = ctx.Err()
		} else {
			v.ObservationID, err = r.collect(ctx, id)
		}
		switch {
		case err == nil:
		case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
			v.Outcome = "cancelled"
			v.ErrorSummary = "collection_cancelled"
		case errors.Is(err, domain.ErrTrackingDisabled):
			v.Outcome = "unavailable"
			v.ErrorSummary = "tracking_disabled"
		case errors.Is(err, ErrUnavailable):
			v.Outcome = "unavailable"
			v.ErrorSummary = "collection_unavailable"
		case errors.Is(err, ErrBusy):
			v.Outcome = "unavailable"
			v.ErrorSummary = "collection_busy"
		case errors.Is(err, sql.ErrNoRows):
			v.Outcome = "unavailable"
			v.ErrorSummary = "listing_not_found"
		default:
			v.Outcome = "failure"
			v.ErrorSummary = "collection_failed"
		}
		results = append(results, v)
	}
	return results, nil
}
