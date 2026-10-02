package collector_test

import (
	"context"
	"errors"
	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"reflect"
	"testing"
	"time"
)

func TestBulkCollection(t *testing.T) {
	m, p := fixtures(t)
	disabled := m.listings["a"]
	disabled.ID = "disabled"
	disabled.TrackingDisabled = true
	m.listings[disabled.ID] = disabled
	s := &auditStore{memoryStore: m}
	r, _ := collector.New(s, []collector.Target{{ListingID: "a", Provider: p}, {ListingID: "bad", Provider: p}, {ListingID: "b", Provider: p}, {ListingID: "disabled", Provider: p}}, time.Hour, time.Second)
	got, err := r.CollectBulk(context.Background(), []string{"a", "bad", "disabled", "unknown", "a", "b"})
	if err != nil || len(got) != 5 {
		t.Fatal(got, err)
	}
	for i, want := range []string{"success", "failure", "unavailable", "unavailable", "success"} {
		if got[i].Outcome != want {
			t.Fatal(got)
		}
	}
	if got[2].ErrorSummary != "tracking_disabled" || got[3].ErrorSummary != "collection_unavailable" || got[1].ErrorSummary != "collection_failed" {
		t.Fatal(got)
	}
	if len(m.writes) != 2 || len(s.attempts) != 3 {
		t.Fatal(m.writes, s.attempts)
	}
	for _, i := range []int{0, 4} {
		if _, ok := m.writes[got[i].ObservationID]; !ok {
			t.Fatal(got)
		}
	}
	for _, a := range s.attempts {
		if a.Trigger != "manual" {
			t.Fatal(a)
		}
	}
	if s.attempts[0].ObservationID != got[0].ObservationID || s.attempts[2].ObservationID != got[4].ObservationID {
		t.Fatal("wrong reference")
	}
}
func TestBulkValidationAndCancellation(t *testing.T) {
	m, p := fixtures(t)
	s := &auditStore{memoryStore: m}
	r, _ := collector.New(s, []collector.Target{{ListingID: "a", Provider: p}}, time.Hour, time.Second)
	for _, ids := range [][]string{nil, {}, {" "}, {"a", ""}, make([]string, 21)} {
		if _, err := r.CollectBulk(context.Background(), ids); !errors.Is(err, collector.ErrInvalidBatch) {
			t.Fatal(ids, err)
		}
	}
	if len(s.attempts) != 0 {
		t.Fatal("validation started work")
	}
	if got, err := collector.BulkIDs([]string{"constructor", "__proto__", "constructor"}); err != nil || !reflect.DeepEqual(got, []string{"constructor", "__proto__"}) {
		t.Fatal(got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancelling := providerFunc(func(c context.Context, l domain.Listing) (domain.PriceObservation, error) {
		cancel()
		return domain.PriceObservation{}, c.Err()
	})
	r, _ = collector.New(s, []collector.Target{{ListingID: "a", Provider: p}, {ListingID: "bad", Provider: cancelling}, {ListingID: "b", Provider: p}}, time.Hour, time.Second)
	got, err := r.CollectBulk(ctx, []string{"a", "bad", "b"})
	if err != nil || got[0].Outcome != "success" || got[1].Outcome != "cancelled" || got[2].Outcome != "cancelled" || len(m.writes) != 1 || len(s.attempts) != 2 {
		t.Fatal(got, err, s.attempts)
	}
	if s.attempts[1].Outcome != "cancelled" {
		t.Fatal(s.attempts)
	}
}
