package main

import (
	"context"
	"database/sql"
	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type trackingStub struct {
	calls   int
	enabled bool
	err     error
}

func (s *trackingStub) SetListingTracking(_ context.Context, _ string, enabled bool) error {
	s.calls++
	s.enabled = enabled
	return s.err
}
func TestTrackingAPI(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `{"tracking_enabled":null}`, `{"tracking_enabled":"false"}`, `{`} {
		s := &trackingStub{}
		w := httptest.NewRecorder()
		trackingHandler(s)(w, httptest.NewRequest("PATCH", "/", strings.NewReader(body)))
		if w.Code != 400 || s.calls != 0 {
			t.Fatal(w.Code, s)
		}
	}
	for _, enabled := range []string{"false", "true"} {
		s := &trackingStub{}
		w := httptest.NewRecorder()
		trackingHandler(s)(w, httptest.NewRequest("PATCH", "/", strings.NewReader(`{"tracking_enabled":`+enabled+`}`)))
		if w.Code != 200 || s.enabled != (enabled == "true") {
			t.Fatal(w.Code, s)
		}
	}
	s := &trackingStub{err: sql.ErrNoRows}
	w := httptest.NewRecorder()
	trackingHandler(s)(w, httptest.NewRequest("PATCH", "/", strings.NewReader(`{"tracking_enabled":false}`)))
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
}
func TestDisabledRefreshAndStatus(t *testing.T) {
	l := domain.Listing{ID: "l", TrackingDisabled: true}
	called := false
	api := currentAPI{store: currentStub{values: []persistence.CurrentListing{{Listing: l}}}, collect: func(context.Context, string) error { called = true; return nil }}
	r := httptest.NewRequest("POST", "/listings/l/collect", nil)
	r.SetPathValue("id", "l")
	w := httptest.NewRecorder()
	api.collectListing(w, r)
	if w.Code != 409 || called || !strings.Contains(w.Body.String(), "tracking_disabled") {
		t.Fatal(w.Code, w.Body)
	}
	got := currentResponse(persistence.CurrentListing{Listing: l}, collector.Status{State: "failed", Error: "collection_failed"}, time.Now(), time.Minute)
	if got.Listing.TrackingEnabled || got.Collection.State != "disabled" || got.Collection.Error != "" || got.Freshness != "missing" {
		t.Fatal(got)
	}
	if !(domain.Listing{}).TrackingEnabled() {
		t.Fatal("zero-value Listing tracking must default enabled")
	}
	summary := summarizeProduct(domain.Product{ID: "p"}, []persistence.CurrentListing{{Listing: l}}, func(string) collector.Status { return collector.Status{State: "failed", Error: "collection_failed"} }, time.Now(), time.Minute)
	if summary.Collection["disabled"] != 1 || summary.HasCollectionError || summary.Freshness["missing"] != 1 {
		t.Fatal("dashboard conflates tracking/error/freshness", summary)
	}
}
