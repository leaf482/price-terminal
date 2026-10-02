package persistence

import (
	"testing"
	"time"
)

func TestCollectionHealthCounts(t *testing.T) {
	now := time.Now()
	rows := []CollectionHealthListing{
		{TrackingEnabled: true, AttemptedAt: &now, Outcome: "success"},
		{TrackingEnabled: true, AttemptedAt: &now, Outcome: "provider_error"},
		{TrackingEnabled: true},
		{TrackingEnabled: false, AttemptedAt: &now, Outcome: "success"},
	}
	got := collectionHealthCounts(rows)
	if got != (CollectionHealthCounts{Total: 4, Healthy: 1, Error: 1, Never: 1, Disabled: 1}) {
		t.Fatal(got)
	}
	if collectionHealthCounts(nil) != (CollectionHealthCounts{}) {
		t.Fatal("empty")
	}
}
