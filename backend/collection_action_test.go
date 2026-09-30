package main

import (
	"context"
	"database/sql"
	"errors"
	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestManualCollectionAPI(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		code int
	}{{"success", nil, 200}, {"failure", errors.New("secret provider trace"), 502}, {"unsupported", collector.ErrUnavailable, 503}, {"busy", collector.ErrBusy, 409}} {
		t.Run(tt.name, func(t *testing.T) {
			old := priceNow.Add(-time.Hour)
			value := currentFixture(t, "l", domain.USD, nil, old)
			calls := 0
			api := currentAPI{store: currentStub{values: []persistence.CurrentListing{value}}, now: func() time.Time { return priceNow }, maxAge: time.Minute, status: func(string) collector.Status {
				return collector.Status{State: "failed", LastAttemptedAt: &priceNow, LastSuccessfulAt: &old, Error: "collection_failed"}
			}, collect: func(_ context.Context, id string) error {
				calls++
				if id != "l" {
					t.Fatal(id)
				}
				return tt.err
			}}
			mux := http.NewServeMux()
			registerCurrentRoutes(mux, api)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("POST", "/listings/l/collect", nil))
			if w.Code != tt.code || calls != 1 || strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w.Code, w.Body.String())
			}
			w = httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", "/listings/l/price", nil))
			body := w.Body.String()
			if w.Code != 200 || !strings.Contains(body, `"freshness":"stale"`) || !strings.Contains(body, `"observed_at":"`+old.Format(time.RFC3339)+`"`) || !strings.Contains(body, `"last_attempted_at":"`+priceNow.Format(time.RFC3339)+`"`) {
				t.Fatal(body)
			}
		})
	}
	for _, missing := range []bool{false, true} {
		s := currentStub{values: []persistence.CurrentListing{currentFixture(t, "l", domain.USD, nil, priceNow)}}
		want := 503
		if missing {
			s.err = sql.ErrNoRows
			want = 404
		}
		mux := http.NewServeMux()
		registerCurrentRoutes(mux, currentAPI{store: s})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("POST", "/listings/l/collect", nil))
		if w.Code != want {
			t.Fatal(w.Code)
		}
	}
}
