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

type auditStore struct {
	*memoryStore
	attempts  []domain.CollectionAttempt
	auditFail bool
}

func (s *auditStore) InsertCollectionAttempt(ctx context.Context, a domain.CollectionAttempt) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.auditFail {
		return errors.New("secret metadata error")
	}
	s.attempts = append(s.attempts, a)
	return nil
}
func (s *auditStore) EvaluateAlerts(context.Context, string) error {
	return errors.New("secret alert error")
}

func TestAttemptRecording(t *testing.T) {
	for _, tc := range []struct {
		name, id, outcome string
		fail              bool
	}{
		{"success despite alert error", "a", "success", false}, {"provider", "bad", "provider_error", false}, {"persistence", "a", "persistence_error", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, p := fixtures(t)
			m.fail = tc.fail
			s := &auditStore{memoryStore: m}
			r, _ := collector.New(s, []collector.Target{{ListingID: tc.id, Provider: p}}, time.Hour, time.Second)
			err := r.Collect(context.Background(), tc.id)
			if (err == nil) != (tc.outcome == "success") || len(s.attempts) != 1 {
				t.Fatal(err, s.attempts)
			}
			a := s.attempts[0]
			if a.Outcome != tc.outcome || a.Trigger != "manual" || a.FinishedAt.Before(a.StartedAt) {
				t.Fatal(a)
			}
			if tc.outcome == "success" {
				if _, ok := m.writes[a.ObservationID]; !ok || a.ErrorSummary != "" {
					t.Fatal(a)
				}
			} else if a.ErrorSummary != tc.outcome || a.ObservationID != "" || len(m.writes) != 0 {
				t.Fatal(a)
			}
		})
	}
}
func TestAttemptCancellationAndRejectedRequests(t *testing.T) {
	m, _ := fixtures(t)
	s := &auditStore{memoryStore: m}
	ctx, cancel := context.WithCancel(context.Background())
	p := providerFunc(func(ctx context.Context, l domain.Listing) (domain.PriceObservation, error) {
		cancel()
		return domain.PriceObservation{}, ctx.Err()
	})
	r, _ := collector.New(s, []collector.Target{{ListingID: "a", Provider: p}}, time.Hour, time.Second)
	if !errors.Is(r.Collect(ctx, "a"), context.Canceled) || len(s.attempts) != 1 || s.attempts[0].Outcome != "cancelled" {
		t.Fatal(s.attempts)
	}
	if !errors.Is(r.Collect(context.Background(), "unconfigured"), collector.ErrUnavailable) {
		t.Fatal("unavailable")
	}
	l := m.listings["a"]
	l.TrackingDisabled = true
	m.listings["a"] = l
	if !errors.Is(r.Collect(context.Background(), "a"), domain.ErrTrackingDisabled) || len(s.attempts) != 1 {
		t.Fatal("rejection recorded")
	}
}
func TestScheduledAttemptsAndMetadataFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m, p := fixtures(t)
		s := &auditStore{memoryStore: m}
		r, _ := collector.New(s, []collector.Target{{ListingID: "bad", Provider: p}, {ListingID: "a", Provider: p}}, time.Hour, time.Second)
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- r.Run(ctx) }()
		synctest.Wait()
		cancel()
		<-done
		if len(s.attempts) != 2 || s.attempts[0].Outcome != "provider_error" || s.attempts[1].Outcome != "success" || s.attempts[1].Trigger != "scheduled" || len(m.writes) != 1 {
			t.Fatal(s.attempts)
		}
		s.auditFail = true
		if err := r.Collect(context.Background(), "a"); err != nil || len(m.writes) != 2 {
			t.Fatal("audit error corrupted success", err)
		}
	})
}
