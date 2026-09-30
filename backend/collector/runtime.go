// Package collector runs one bounded set of explicitly enabled listings.
package collector

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"github.com/leaf482/price-terminal/backend/provider"
)

type Target struct {
	ListingID string
	Provider  provider.Provider
}

// Status is operational, process-local state, never an observation. Restart
// resets it. Error codes deliberately exclude provider/DB messages and secrets.
type Status struct {
	Active              bool       `json:"active"`
	State               string     `json:"state"`
	LastAttemptedAt     *time.Time `json:"last_attempted_at,omitempty"`
	LastSuccessfulAt    *time.Time `json:"last_successful_at,omitempty"`
	Error               string     `json:"error,omitempty"`
	ConsecutiveFailures uint64     `json:"consecutive_failures"`
}

type Runtime struct {
	store             ingestion.Store
	targets           []Target
	interval, timeout time.Duration
	running           atomic.Bool
	mu                sync.RWMutex
	states            map[string]Status
}

func New(store ingestion.Store, targets []Target, interval, timeout time.Duration) (*Runtime, error) {
	if store == nil || interval <= 0 || timeout <= 0 || len(targets) > ingestion.MaxBatchSize {
		return nil, fmt.Errorf("collector: invalid configuration")
	}
	r := &Runtime{store: store, targets: append([]Target(nil), targets...), interval: interval, timeout: timeout, states: make(map[string]Status)}
	for _, target := range targets {
		if strings.TrimSpace(target.ListingID) == "" || target.Provider == nil {
			return nil, fmt.Errorf("collector: listing and provider required")
		}
		if _, exists := r.states[target.ListingID]; exists {
			return nil, fmt.Errorf("collector: duplicate listing")
		}
		r.states[target.ListingID] = Status{Active: true, State: "never_attempted"}
	}
	return r, nil
}

func (r *Runtime) Status(id string) Status {
	if r == nil {
		return Status{State: "inactive"}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.states[id]
	if !ok {
		return Status{State: "inactive"}
	}
	// Return independent timestamp pointers so callers cannot mutate shared state.
	if s.LastAttemptedAt != nil {
		v := *s.LastAttemptedAt
		s.LastAttemptedAt = &v
	}
	if s.LastSuccessfulAt != nil {
		v := *s.LastSuccessfulAt
		s.LastSuccessfulAt = &v
	}
	return s
}

func (r *Runtime) start(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.states[id]
	if s.State == "collecting" {
		return false
	}
	now := time.Now().UTC()
	s.LastAttemptedAt = &now
	s.State = "collecting"
	s.Error = ""
	r.states[id] = s
	slog.Info("collection_start", "listing_id", id)
	return true
}

var ErrUnavailable = errors.New("no provider configured for listing")
var ErrBusy = errors.New("listing collection already in progress")

// Collect performs one configured attempt synchronously. The same per-listing
// guard is used by scheduled cycles; no work is queued or retried.
func (r *Runtime) Collect(ctx context.Context, id string) (err error) {
	if r == nil {
		return ErrUnavailable
	}
	var selected *Target
	for i := range r.targets {
		if r.targets[i].ListingID == id {
			selected = &r.targets[i]
			break
		}
	}
	if selected == nil {
		return ErrUnavailable
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	listing, err := r.store.GetListing(ctx, id)
	if err != nil {
		if r.start(id) {
			r.finish(id, err)
		}
		return err
	}
	if listing.TrackingDisabled {
		return domain.ErrTrackingDisabled
	}
	if !r.start(id) {
		return ErrBusy
	}
	defer func() { r.finish(id, err) }()
	_, err = ingestion.New(selected.Provider, r.store).Ingest(ctx, rand.Text(), listing)
	return err
}
func (r *Runtime) finish(id string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.states[id]
	if errors.Is(err, domain.ErrTrackingDisabled) {
		s.State = "disabled"
		s.Error = ""
	} else if err != nil {
		s.State = "failed"
		s.Error = "collection_failed"
		s.ConsecutiveFailures++
	} else {
		now := time.Now().UTC()
		s.LastSuccessfulAt = &now
		s.State = "success"
		s.Error = ""
		s.ConsecutiveFailures = 0
	}
	r.states[id] = s
	slog.Info("collection_result", "listing_id", id, "state", s.State, "error_code", s.Error, "consecutive_failures", s.ConsecutiveFailures)
}

// Run collects immediately, then waits interval AFTER each completed cycle.
// There is no overlap, catch-up, retry, or detached per-listing goroutine.
// A second Run on the same instance is rejected. Only one instance is supported.
func (r *Runtime) Run(ctx context.Context) error {
	if !r.running.CompareAndSwap(false, true) {
		return fmt.Errorf("collector: already running")
	}
	defer r.running.Store(false)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.cycle(ctx)
		timer := time.NewTimer(r.interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (r *Runtime) cycle(ctx context.Context) {
	jobs := make([]ingestion.Job, 0, len(r.targets))
	for _, target := range r.targets {
		if ctx.Err() != nil {
			break
		}
		// Source resolution is part of this attempt; missing listings remain errors.
		lookupCtx, cancel := context.WithTimeout(ctx, r.timeout)
		listing, err := r.store.GetListing(lookupCtx, target.ListingID)
		cancel()
		if err != nil {
			if r.start(target.ListingID) {
				r.finish(target.ListingID, err)
			}
			continue
		}
		if listing.TrackingDisabled {
			continue
		}
		if !r.start(target.ListingID) {
			continue
		}
		jobs = append(jobs, ingestion.Job{
			Listing: listing, ResultID: rand.Text(), Provider: target.Provider,
			OnComplete: func(outcome ingestion.Outcome) {
				r.finish(outcome.Listing.ID, outcome.Err)
			},
		})
	}
	outcomes, err := ingestion.Run(ctx, r.store, jobs, r.timeout)
	if err != nil && len(outcomes) == 0 {
		for _, job := range jobs {
			r.finish(job.Listing.ID, err)
		}
		return
	}
}
