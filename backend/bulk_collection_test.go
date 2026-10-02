package main

import (
	"context"
	"github.com/leaf482/price-terminal/backend/collector"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBulkCollectionAPI(t *testing.T) {
	calls := 0
	handler := bulkCollectionHandler(func(ctx context.Context, ids []string) ([]collector.BulkResult, error) {
		calls++
		if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
			t.Fatal(ids)
		}
		return []collector.BulkResult{{ListingID: "a", Outcome: "success", ObservationID: "observation"}, {ListingID: "b", Outcome: "failure", ErrorSummary: "collection_failed"}}, nil
	})
	for _, body := range []string{`{`, `null`, `{}`, `{"listing_ids":[]}`, `{"listing_ids":[null]}`, `{"listing_ids":["a",""]}`, `{"listing_ids":[` + strings.Repeat(`"a",`, 20) + `"b"]}`} {
		w := httptest.NewRecorder()
		handler(w, httptest.NewRequest("POST", "/collection/refresh", strings.NewReader(body)))
		if w.Code != 400 || calls != 0 {
			t.Fatal(w.Code, calls)
		}
	}
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest("POST", "/collection/refresh", strings.NewReader(`{"listing_ids":["a","b","a"]}`)))
	if w.Code != 200 || calls != 1 || !strings.Contains(w.Body.String(), `"observation_id":"observation"`) || !strings.Contains(w.Body.String(), `"outcome":"failure"`) {
		t.Fatal(w.Body.String())
	}
	w = httptest.NewRecorder()
	bulkCollectionHandler(nil)(w, httptest.NewRequest("POST", "/collection/refresh", strings.NewReader(`{"listing_ids":["a"]}`)))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "unavailable") {
		t.Fatal(w.Body.String())
	}
}
