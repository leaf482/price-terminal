package ingestion_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"github.com/leaf482/price-terminal/backend/provider"
)

func batchFixtures(t *testing.T, n int) ([]ingestion.Job, storeStub, map[string]int) {
	t.Helper()
	jobs := make([]ingestion.Job, n)
	listings := make(map[string]domain.Listing)
	writes := make(map[string]int)
	for index := range jobs {
		l := listing()
		l.ID = fmt.Sprintf("l%d", index)
		l.URL += "/" + l.ID
		o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: l.ID, ObservedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Source: l.URL, Stock: domain.StockUnknown})
		if err != nil {
			t.Fatal(err)
		}
		jobs[index] = ingestion.Job{Listing: l, ResultID: "result-" + l.ID, Provider: provider.NewFake(map[string]provider.FakeResponse{l.ID: {Observation: o}})}
		listings[l.ID] = l
	}
	store := storeStub{get: func(_ context.Context, id string) (domain.Listing, error) { return listings[id], nil }, insert: func(_ context.Context, id string, o domain.PriceObservation) error {
		if id != "result-"+o.ListingID() {
			t.Fatal("incorrect result association")
		}
		writes[id]++
		return nil
	}}
	return jobs, store, writes
}

func TestBatchFailureIsolation(t *testing.T) {
	for _, failures := range [][]int{nil, {0}, {0, 2}} {
		jobs, store, writes := batchFixtures(t, 4)
		cause := errors.New("source unavailable")
		failed := map[int]bool{}
		for _, index := range failures {
			failed[index] = true
			jobs[index].Provider = provider.NewFake(map[string]provider.FakeResponse{jobs[index].Listing.ID: {Err: cause}})
		}
		results, err := ingestion.Run(context.Background(), store, jobs, time.Second)
		if err != nil || len(results) != len(jobs) {
			t.Fatalf("results=%v err=%v", results, err)
		}
		for index, result := range results {
			if result.Listing != jobs[index].Listing || result.ResultID != jobs[index].ResultID || !result.Started {
				t.Fatal("association/order changed")
			}
			if failed[index] {
				if !errors.Is(result.Err, cause) || writes[result.ResultID] != 0 {
					t.Fatal("collection failure persisted or cause lost")
				}
			} else if result.Err != nil || writes[result.ResultID] != 1 || result.Collected.ID() != result.ResultID {
				t.Fatalf("healthy result=%+v writes=%v", result, writes)
			}
		}
	}
}

func TestBatchPersistenceFailureContinues(t *testing.T) {
	jobs, store, writes := batchFixtures(t, 3)
	cause := errors.New("write failed")
	insert := store.insert
	store.insert = func(c context.Context, id string, o domain.PriceObservation) error {
		if id == jobs[1].ResultID {
			return cause
		}
		return insert(c, id, o)
	}
	results, err := ingestion.Run(context.Background(), store, jobs, time.Second)
	if err != nil || !errors.Is(results[1].Err, cause) || results[1].Collected.ID() != jobs[1].ResultID || len(writes) != 2 {
		t.Fatalf("results=%v err=%v writes=%v", results, err, writes)
	}
}

func TestBatchCancellation(t *testing.T) {
	jobs, store, writes := batchFixtures(t, 4)
	ctx, cancel := context.WithCancel(context.Background())
	jobs[1].Provider = providerFunc(func(c context.Context, _ domain.Listing) (domain.PriceObservation, error) {
		cancel()
		return domain.PriceObservation{}, c.Err()
	})
	results, err := ingestion.Run(ctx, store, jobs, time.Second)
	if !errors.Is(err, context.Canceled) || len(writes) != 1 || results[0].Err != nil {
		t.Fatalf("results=%v writes=%v err=%v", results, writes, err)
	}
	for index := 1; index < len(jobs); index++ {
		if !errors.Is(results[index].Err, context.Canceled) || results[index].Started != (index == 1) {
			t.Fatalf("cancellation outcome=%+v", results[index])
		}
	}
	results, err = ingestion.Run(ctx, nil, jobs, time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Started || !errors.Is(result.Err, context.Canceled) {
			t.Fatal("pre-canceled batch started work")
		}
	}
}

func TestBatchItemTimeoutContinues(t *testing.T) {
	// Virtual time avoids sleeps and timing-sensitive assertions.
	synctest.Test(t, func(t *testing.T) {
		jobs, store, writes := batchFixtures(t, 2)
		jobs[0].Provider = providerFunc(func(c context.Context, _ domain.Listing) (domain.PriceObservation, error) {
			<-c.Done()
			return domain.PriceObservation{}, c.Err()
		})
		results, err := ingestion.Run(context.Background(), store, jobs, time.Second)
		if err != nil || !errors.Is(results[0].Err, context.DeadlineExceeded) || results[1].Err != nil || len(writes) != 1 {
			t.Fatalf("results=%v writes=%v err=%v", results, writes, err)
		}
	})
}

func TestBatchBoundsAndDuplicateRejection(t *testing.T) {
	jobs, _, _ := batchFixtures(t, 2)
	for _, test := range []struct {
		jobs    []ingestion.Job
		timeout time.Duration
	}{
		{make([]ingestion.Job, ingestion.MaxBatchSize+1), time.Second},
		{jobs, 0}, {jobs, -time.Second}, {[]ingestion.Job{jobs[0], jobs[0]}, time.Second},
		{[]ingestion.Job{jobs[0], {Listing: jobs[1].Listing, ResultID: jobs[0].ResultID}}, time.Second},
	} {
		if results, err := ingestion.Run(context.Background(), nil, test.jobs, test.timeout); err == nil || results != nil {
			t.Fatal("invalid batch accepted")
		}
	}
	if results, err := ingestion.Run(context.Background(), nil, nil, time.Second); err != nil || len(results) != 0 {
		t.Fatal("empty batch failed")
	}
	maximum, store, writes := batchFixtures(t, ingestion.MaxBatchSize)
	if results, err := ingestion.Run(context.Background(), store, maximum, time.Second); err != nil || len(results) != ingestion.MaxBatchSize || len(writes) != ingestion.MaxBatchSize {
		t.Fatal("maximum batch failed")
	}
}

func TestBatchInvalidItemContinues(t *testing.T) {
	jobs, store, writes := batchFixtures(t, 3)
	jobs[0].Provider = nil
	jobs[1].Listing.URL = "/invalid"
	results, err := ingestion.Run(context.Background(), store, jobs, time.Second)
	if err != nil || results[0].Err == nil || results[1].Err == nil || results[2].Err != nil || len(writes) != 1 {
		t.Fatalf("results=%v err=%v", results, err)
	}
}
