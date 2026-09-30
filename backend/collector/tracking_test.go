package collector_test

import (
	"context"
	"errors"
	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"testing"
	"testing/synctest"
	"time"
)

func TestTrackingCollection(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, fake := fixtures(t)
		listing := s.listings["a"]
		listing.TrackingDisabled = true
		s.listings["a"] = listing
		calls := 0
		p := providerFunc(func(c context.Context, l domain.Listing) (domain.PriceObservation, error) {
			calls++
			return fake.Collect(c, l)
		})
		r, err := collector.New(s, []collector.Target{{ListingID: "a", Provider: p}}, time.Minute, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Collect(context.Background(), "a"); !errors.Is(err, domain.ErrTrackingDisabled) {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- r.Run(ctx) }()
		synctest.Wait()
		if calls != 0 || len(s.writes) != 0 || r.Status("a").LastAttemptedAt != nil {
			t.Fatal("disabled collection attempted")
		}
		listing.TrackingDisabled = false
		s.listings["a"] = listing
		time.Sleep(time.Minute)
		synctest.Wait()
		if calls != 1 || len(s.writes) != 1 {
			t.Fatal("re-enable did not collect")
		}
		cancel()
		<-done
		if err := r.Collect(context.Background(), "a"); err != nil {
			t.Fatal(err)
		}
		listing.TrackingDisabled = true
		s.listings["a"] = listing
		o, err := fake.Collect(context.Background(), listing)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ingestion.New(nil, s).Record(context.Background(), "manual", listing, o); err != nil {
			t.Fatal("manual entry blocked", err)
		}
	})
}

func TestDisableDuringCollection(t *testing.T) {
	s, fake := fixtures(t)
	p := providerFunc(func(ctx context.Context, l domain.Listing) (domain.PriceObservation, error) {
		stored := s.listings[l.ID]
		stored.TrackingDisabled = true
		s.listings[l.ID] = stored
		return fake.Collect(ctx, l)
	})
	r, _ := collector.New(s, []collector.Target{{ListingID: "a", Provider: p}}, time.Minute, time.Second)
	if err := r.Collect(context.Background(), "a"); !errors.Is(err, domain.ErrTrackingDisabled) {
		t.Fatal(err)
	}
	if len(s.writes) != 0 || r.Status("a").State != "disabled" || r.Status("a").ConsecutiveFailures != 0 {
		t.Fatal("disabled attempt created facts/error")
	}
}

func TestQueuedListingDisabledBeforeProviderCall(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, fake := fixtures(t)
		calls := []string{}
		p := providerFunc(func(ctx context.Context, l domain.Listing) (domain.PriceObservation, error) {
			calls = append(calls, l.ID)
			if l.ID == "a" {
				b := s.listings["b"]
				b.TrackingDisabled = true
				s.listings["b"] = b
			}
			return fake.Collect(ctx, l)
		})
		r, _ := collector.New(s, []collector.Target{{ListingID: "a", Provider: p}, {ListingID: "b", Provider: p}}, time.Hour, time.Second)
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- r.Run(ctx) }()
		synctest.Wait()
		cancel()
		<-done
		if len(calls) != 1 || calls[0] != "a" || len(s.writes) != 1 || r.Status("b").State != "disabled" {
			t.Fatal("queued disabled Listing collected", calls)
		}
	})
}
