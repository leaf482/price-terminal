package collector_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"github.com/leaf482/price-terminal/backend/provider"
)

type memoryStore struct {
	listings map[string]domain.Listing
	writes   map[string]domain.PriceObservation
	fail     bool
}

func (s *memoryStore) GetListing(ctx context.Context, id string) (domain.Listing, error) {
	if err := ctx.Err(); err != nil {
		return domain.Listing{}, err
	}
	l, ok := s.listings[id]
	if !ok {
		return l, sql.ErrNoRows
	}
	return l, nil
}
func (s *memoryStore) InsertPriceObservation(ctx context.Context, id string, o domain.PriceObservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.fail {
		return errors.New("secret database error")
	}
	if _, ok := s.writes[id]; ok {
		return errors.New("duplicate result")
	}
	s.writes[id] = o
	return nil
}

type providerFunc func(context.Context, domain.Listing) (domain.PriceObservation, error)

func (f providerFunc) Collect(c context.Context, l domain.Listing) (domain.PriceObservation, error) {
	return f(c, l)
}

func fixtures(t *testing.T) (*memoryStore, *provider.Fake) {
	t.Helper()
	s := &memoryStore{listings: map[string]domain.Listing{}, writes: map[string]domain.PriceObservation{}}
	responses := map[string]provider.FakeResponse{}
	for _, id := range []string{"a", "b", "bad"} {
		l := domain.Listing{ID: id, ProductID: "p", RetailerID: "r", URL: "https://example.com/" + id}
		s.listings[id] = l
		var price *domain.Money
		if id == "a" {
			price = &domain.Money{Currency: domain.USD, MinorUnits: 0}
		}
		o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: id, ObservedAt: time.Date(2020, 1, 1, 0, 0, 0, 1, time.UTC), Source: "fixture", Stock: domain.StockUnknown, OfferPrice: price})
		if err != nil {
			t.Fatal(err)
		}
		responses[id] = provider.FakeResponse{Observation: o}
	}
	responses["bad"] = provider.FakeResponse{Err: errors.New("secret provider error")}
	return s, provider.NewFake(responses)
}

func TestPeriodicCollectionAndStatus(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, p := fixtures(t)
		targets := []collector.Target{{ListingID: "bad", Provider: p}, {ListingID: "missing", Provider: p}, {ListingID: "a", Provider: p}, {ListingID: "b", Provider: p}}
		r, err := collector.New(s, targets, time.Second, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if r.Status("a").State != "never_attempted" || r.Status("other").Active {
			t.Fatal("initial status")
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- r.Run(ctx) }()
		synctest.Wait()
		if len(s.writes) != 2 {
			t.Fatalf("writes=%v", s.writes)
		}
		first := r.Status("a")
		if first.State != "success" || first.LastSuccessfulAt == nil || first.LastAttemptedAt == nil {
			t.Fatal(first)
		}
		original := *first.LastSuccessfulAt
		*first.LastSuccessfulAt = time.Time{}
		if r.Status("a").LastSuccessfulAt.IsZero() {
			t.Fatal("mutable status leaked")
		}
		for _, id := range []string{"bad", "missing"} {
			status := r.Status(id)
			if status.State != "failed" || status.Error != "collection_failed" || status.LastSuccessfulAt != nil {
				t.Fatal(status)
			}
		}
		time.Sleep(time.Second)
		synctest.Wait()
		if len(s.writes) != 4 {
			t.Fatal("independent cycle did not append exactly twice")
		}
		for _, o := range s.writes {
			if o.ObservedAt().Year() != 2020 {
				t.Fatal("fixture timestamp refreshed")
			}
			if o.ListingID() == "bad" {
				t.Fatal("failure persisted")
			}
		}
		last := *r.Status("a").LastSuccessfulAt
		if !last.After(original) {
			t.Fatal("success time did not advance")
		}
		s.fail = true
		time.Sleep(time.Second)
		synctest.Wait()
		if r.Status("a").State != "failed" || !r.Status("a").LastSuccessfulAt.Equal(last) || len(s.writes) != 4 {
			t.Fatal("failure erased success or persisted")
		}
		cancel()
		synctest.Wait()
		if !errors.Is(<-done, context.Canceled) {
			t.Fatal("cancellation lost")
		}
	})
}

func TestNoOverlapAndCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, _ := fixtures(t)
		calls := 0
		p := providerFunc(func(ctx context.Context, _ domain.Listing) (domain.PriceObservation, error) {
			calls++
			<-ctx.Done()
			return domain.PriceObservation{}, ctx.Err()
		})
		r, err := collector.New(s, []collector.Target{{ListingID: "a", Provider: p}}, time.Second, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- r.Run(ctx) }()
		synctest.Wait()
		time.Sleep(5 * time.Second)
		synctest.Wait()
		if calls != 1 {
			t.Fatal("cycles overlapped")
		}
		if err := r.Run(ctx); err == nil {
			t.Fatal("second runner accepted")
		}
		cancel()
		synctest.Wait()
		if !errors.Is(<-done, context.Canceled) || len(s.writes) != 0 || r.Status("a").State != "failed" {
			t.Fatal("shutdown failed")
		}
	})
}

func TestConfigurationBounds(t *testing.T) {
	s, p := fixtures(t)
	for _, targets := range [][]collector.Target{{{ListingID: "", Provider: p}}, {{ListingID: "a"}}, {{ListingID: "a", Provider: p}, {ListingID: "a", Provider: p}}, make([]collector.Target, ingestion.MaxBatchSize+1)} {
		if _, err := collector.New(s, targets, time.Second, time.Second); err == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
	for _, duration := range []time.Duration{0, -1} {
		if _, err := collector.New(s, nil, duration, time.Second); err == nil {
			t.Fatal("invalid interval")
		}
	}
	r, _ := collector.New(s, nil, time.Second, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(fmt.Sprint(err))
	}
}

func TestDeadlineDoesNotStopHealthyListing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, fake := fixtures(t)
		blocked := providerFunc(func(ctx context.Context, _ domain.Listing) (domain.PriceObservation, error) {
			<-ctx.Done()
			return domain.PriceObservation{}, ctx.Err()
		})
		r, err := collector.New(s, []collector.Target{{ListingID: "bad", Provider: blocked}, {ListingID: "a", Provider: fake}}, time.Hour, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- r.Run(ctx) }()
		synctest.Wait()
		time.Sleep(time.Second)
		synctest.Wait()
		if r.Status("bad").State != "failed" || r.Status("a").State != "success" || len(s.writes) != 1 {
			t.Fatal("deadline prevented healthy work")
		}
		cancel()
		synctest.Wait()
		<-done
	})
}
