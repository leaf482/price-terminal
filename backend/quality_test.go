package main

import (
	"context"
	"database/sql"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type qualityStub struct {
	value  persistence.ObservationAudit
	err    error
	writes int
}

func (s *qualityStub) GetListing(context.Context, string) (domain.Listing, error) {
	return domain.Listing{ID: "l"}, s.err
}
func (s *qualityStub) GetObservationAudit(context.Context, string) (persistence.ObservationAudit, error) {
	return s.value, s.err
}
func (s *qualityStub) ListObservationAudit(context.Context, string) ([]persistence.ObservationAudit, bool, error) {
	return []persistence.ObservationAudit{s.value}, false, s.err
}
func (s *qualityStub) InvalidateObservation(_ context.Context, _ string, reason string) (persistence.ObservationAudit, error) {
	s.writes++
	if s.err == nil && s.value.InvalidatedAt == nil {
		at := time.Unix(5, 0).UTC()
		s.value.InvalidatedAt = &at
		s.value.Reason = reason
	}
	return s.value, s.err
}
func TestQualityAPI(t *testing.T) {
	o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: time.Unix(1, 0), Source: "evidence", Stock: domain.StockUnknown})
	if err != nil {
		t.Fatal(err)
	}
	s := &qualityStub{value: persistence.ObservationAudit{ID: "o", Observation: o}}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		mux := http.NewServeMux()
		registerQualityRoutes(mux, s)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		return w
	}
	for _, body := range []string{`{`, `null`, `{}`, `{"reason":null}`, `{"reason":" "}`, `{"reason":"` + strings.Repeat("a", 1001) + `"}`} {
		w := request("POST", "/observations/o/invalidate", body)
		if w.Code != 400 || s.writes != 0 {
			t.Fatal(w.Code, s.writes)
		}
	}
	w := request("GET", "/listings/l/observations", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"valid":true`) {
		t.Fatal(w.Body.String())
	}
	w = request("POST", "/observations/o/invalidate", `{"reason":"wrong price"}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"valid":false`) {
		t.Fatal(w.Body.String())
	}
	w = request("GET", "/observations/o", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "wrong price") || !strings.Contains(w.Body.String(), "evidence") {
		t.Fatal(w.Body.String())
	}
	for _, test := range []struct {
		err    error
		status int
	}{{sql.ErrNoRows, 404}, {errors.New("secret driver data"), 500}} {
		s.err = test.err
		for _, path := range []string{"/observations/absent", "/listings/absent/observations"} {
			w = request("GET", path, "")
			if w.Code != test.status || strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w.Code, w.Body.String())
			}
		}
		w = request("POST", "/observations/absent/invalidate", `{"reason":"invalid"}`)
		if w.Code != test.status {
			t.Fatal(w.Code)
		}
	}
}
