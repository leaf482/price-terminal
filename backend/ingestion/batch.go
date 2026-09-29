package ingestion

import (
	"context"
	"fmt"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/provider"
)

const MaxBatchSize = 100

// Job identifies one independent collection. ResultID follows Ingest's identity
// rules. Each job can select its own provider without a registry or plugin system.
type Job struct {
	Listing  domain.Listing
	ResultID string
	Provider provider.Provider
}

// Outcome corresponds to the input job at the same index. Err == nil indicates
// a successful write. Started is false when parent cancellation prevented an
// attempt. Collected is retained on persistence failure for an explicit Persist
// retry; it is not itself evidence of a successful write.
type Outcome struct {
	Listing   domain.Listing
	ResultID  string
	Started   bool
	Collected Collected
	Err       error
}

// Run processes at most MaxBatchSize jobs sequentially, with one child deadline
// per job covering collection and persistence. Each attempted job uses Ingest
// once; errors and child timeouts never abort unrelated jobs. Providers and stores
// must honor context, as their existing contracts require; no detached goroutines
// or forced termination are used to work around a non-cooperative implementation.
//
// Invalid batch configuration (size, timeout, repeated Listing/Result IDs) fails
// before any work. Otherwise outcomes always match input order and length. The
// returned error is reserved for parent cancellation; item errors live in outcomes.
// Cancellation preserves completed outcomes and marks unstarted jobs with ctx.Err().
// There are no automatic retries or persistent operational records.
func Run(ctx context.Context, store Store, jobs []Job, perListingTimeout time.Duration) ([]Outcome, error) {
	if len(jobs) > MaxBatchSize {
		return nil, fmt.Errorf("ingest batch: maximum size is %d", MaxBatchSize)
	}
	if perListingTimeout <= 0 {
		return nil, fmt.Errorf("ingest batch: per-listing timeout must be positive")
	}
	listingIDs := make(map[string]bool, len(jobs))
	resultIDs := make(map[string]bool, len(jobs))
	for _, job := range jobs {
		if listingIDs[job.Listing.ID] || resultIDs[job.ResultID] {
			return nil, fmt.Errorf("ingest batch: repeated listing or result ID")
		}
		listingIDs[job.Listing.ID] = true
		resultIDs[job.ResultID] = true
	}
	outcomes := make([]Outcome, len(jobs))
	for index, job := range jobs {
		outcome := Outcome{Listing: job.Listing, ResultID: job.ResultID}
		if err := ctx.Err(); err != nil {
			outcome.Err = err
		} else {
			outcome.Started = true
			if job.Provider == nil {
				outcome.Err = fmt.Errorf("ingest batch: provider is required")
			} else {
				itemCtx, cancel := context.WithTimeout(ctx, perListingTimeout)
				outcome.Collected, outcome.Err = New(job.Provider, store).Ingest(itemCtx, job.ResultID, job.Listing)
				cancel()
			}
		}
		outcomes[index] = outcome
	}
	return outcomes, ctx.Err()
}
