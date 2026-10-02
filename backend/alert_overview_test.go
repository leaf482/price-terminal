package main

import (
	"context"
	"errors"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http/httptest"
	"strings"
	"testing"
)

type alertOverviewStub struct {
	calls int
	err   error
}

func (s *alertOverviewStub) AlertOverview(context.Context) (persistence.AlertOverview, error) {
	s.calls++
	return persistence.AlertOverview{Alerts: []persistence.OverviewAlert{}, Events: []persistence.OverviewEvent{}}, s.err
}
func TestAlertOverviewEndpoint(t *testing.T) {
	for _, fail := range []bool{false, true} {
		s := &alertOverviewStub{}
		if fail {
			s.err = errors.New("secret")
		}
		w := httptest.NewRecorder()
		alertOverviewHandler(s)(w, httptest.NewRequest("GET", "/alerts/overview", nil))
		want := 200
		if fail {
			want = 500
		}
		if w.Code != want || s.calls != 1 || strings.Contains(w.Body.String(), "secret") {
			t.Fatal(w, s)
		}
		if !fail && !strings.Contains(w.Body.String(), `"alerts":[]`) {
			t.Fatal(w)
		}
	}
}
