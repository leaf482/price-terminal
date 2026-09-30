package collector_test

import (
	"context"
	"errors"
	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"testing"
	"testing/synctest"
	"time"
)

func TestManualCollection(t *testing.T) {
	s, p := fixtures(t)
	r, err := collector.New(s, []collector.Target{{ListingID: "a", Provider: p}, {ListingID: "bad", Provider: p}}, time.Hour, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Collect(context.Background(), "unconfigured"); !errors.Is(err, collector.ErrUnavailable) {
		t.Fatal(err)
	}
	if len(s.writes) != 0 {
		t.Fatal("unconfigured wrote")
	}
	if err = r.Collect(context.Background(), "a"); err != nil {
		t.Fatal(err)
	}
	if len(s.writes) != 1 || r.Status("a").State != "success" {
		t.Fatal("not persisted")
	}
	var at time.Time
	for _, o := range s.writes {
		at = o.ObservedAt()
	}
	s.fail = true
	if err = r.Collect(context.Background(), "a"); err == nil {
		t.Fatal("accepted failed persistence")
	}
	status := r.Status("a")
	if status.State != "failed" || status.LastSuccessfulAt == nil || status.Error != "collection_failed" {
		t.Fatal(status)
	}
	for _, o := range s.writes {
		if o.ObservedAt() != at {
			t.Fatal("timestamp changed")
		}
	}
	if err = r.Collect(context.Background(), "bad"); err == nil || len(s.writes) != 1 {
		t.Fatal("provider failure wrote")
	}
	c, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(r.Collect(c, "a"), context.Canceled) {
		t.Fatal("cancellation lost")
	}
}

func TestManualAndScheduledAttemptsCannotOverlap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, fake := fixtures(t)
		release := make(chan struct{})
		calls := 0
		p := providerFunc(func(c context.Context, l domain.Listing) (domain.PriceObservation, error) {
			calls++
			select {
			case <-release:
				return fake.Collect(c, l)
			case <-c.Done():
				return domain.PriceObservation{}, c.Err()
			}
		})
		r, _ := collector.New(s, []collector.Target{{ListingID: "a", Provider: p}}, time.Hour, time.Minute)
		done := make(chan error, 1)
		go func() { done <- r.Collect(context.Background(), "a") }()
		synctest.Wait()
		if !errors.Is(r.Collect(context.Background(), "a"), collector.ErrBusy) {
			t.Fatal("concurrent manual accepted")
		}
		ctx, cancel := context.WithCancel(context.Background())
		scheduled := make(chan error, 1)
		go func() { scheduled <- r.Run(ctx) }()
		synctest.Wait()
		if calls != 1 {
			t.Fatal("schedule overlapped manual")
		}
		close(release)
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		cancel()
		<-scheduled
		if len(s.writes) != 1 {
			t.Fatal("duplicate writes")
		}
	})
}
