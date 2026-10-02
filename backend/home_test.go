package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http/httptest"
	"testing"
	"time"
)

type homeStub struct {
	calls int
	fail  bool
}

func (s *homeStub) HomeCounts(context.Context, time.Time) (persistence.HomeCounts, error) {
	s.calls++
	return persistence.HomeCounts{ActiveProducts: 2, ArchivedProducts: 1}, nil
}
func (s *homeStub) RecentPriceChanges(context.Context) (persistence.PriceChanges, error) {
	s.calls++
	if s.fail {
		return persistence.PriceChanges{}, errors.New("secret")
	}
	v := persistence.PriceChanges{}
	for _, id := range []string{"z", "y", "x", "w", "v", "u"} {
		v.Changes = append(v.Changes, persistence.PriceChange{ListingID: id})
	}
	return v, nil
}
func (s *homeStub) AlertOverview(context.Context) (persistence.AlertOverview, error) {
	s.calls++
	v := persistence.AlertOverview{}
	for _, id := range []string{"z", "y", "x", "w", "v", "u"} {
		v.Events = append(v.Events, persistence.OverviewEvent{ListingID: id})
	}
	return v, nil
}
func (s *homeStub) RecentCollectionFailures(context.Context) ([]persistence.HomeFailure, error) {
	s.calls++
	return []persistence.HomeFailure{}, nil
}
func TestHomeOverview(t *testing.T) {
	for _, fail := range []bool{false, true} {
		s := &homeStub{fail: fail}
		w := httptest.NewRecorder()
		homeHandler(s)(w, httptest.NewRequest("GET", "/overview", nil))
		var body struct {
			Data struct {
				Counts  persistence.HomeCounts      `json:"counts"`
				Changes []persistence.PriceChange   `json:"price_changes"`
				Events  []persistence.OverviewEvent `json:"alert_events"`
				Errors  []string                    `json:"errors"`
			}
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || s.calls != 4 || body.Data.Counts.ActiveProducts != 2 || len(body.Data.Events) != 5 || body.Data.Events[0].ListingID != "z" {
			t.Fatal(w.Body.String(), s.calls)
		}
		if fail {
			if len(body.Data.Errors) != 1 || body.Data.Errors[0] != "price_changes" || body.Data.Changes != nil {
				t.Fatal(w.Body.String())
			}
		} else if len(body.Data.Changes) != 5 || body.Data.Changes[4].ListingID != "v" {
			t.Fatal(w.Body.String())
		}
	}
}
