package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type alertStub struct {
	alerts  []domain.PriceAlert
	err     error
	missing bool
	writes  int
}

func (s *alertStub) GetListing(context.Context, string) (domain.Listing, error) {
	if s.missing {
		return domain.Listing{}, sql.ErrNoRows
	}
	return domain.Listing{ID: "l"}, s.err
}
func (s *alertStub) InsertAlert(_ context.Context, a domain.PriceAlert) error {
	s.writes++
	if s.missing {
		return sql.ErrNoRows
	}
	if s.err != nil {
		return s.err
	}
	s.alerts = append(s.alerts, a)
	return nil
}
func (s *alertStub) ListAlerts(context.Context, string) ([]domain.PriceAlert, error) {
	return s.alerts, s.err
}
func (s *alertStub) SetAlertEnabled(_ context.Context, l, id string, b bool) (domain.PriceAlert, error) {
	if s.err != nil {
		return domain.PriceAlert{}, s.err
	}
	for i, a := range s.alerts {
		if a.ID == id && a.ListingID == l {
			s.alerts[i].Enabled = b
			return s.alerts[i], nil
		}
	}
	return domain.PriceAlert{}, sql.ErrNoRows
}
func (s *alertStub) ListAlertEvents(context.Context, string) ([]persistence.AlertEvent, bool, error) {
	return []persistence.AlertEvent{{AlertID: "a", ListingID: "l", ObservationID: "o", Kind: "target", MinorUnits: 0, Currency: domain.USD, PriceBasis: "offer_price"}}, false, s.err
}
func alertRequest(s *alertStub, method, path, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	registerAlertRoutes(mux, s)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}
func TestAlertAPI(t *testing.T) {
	for _, body := range []string{`{`, `null`, `{}`, `{"kind":"target","currency":"USD"}`, `{"kind":"target","currency":"USD","threshold_minor_units":null}`, `{"kind":"target","currency":"USD","threshold_minor_units":-1}`, `{"kind":"drop","currency":"USD","drop_basis_points":0}`, `{"kind":"historical_low","currency":"EUR"}`, `{"kind":"historical_low","currency":"USD","unexpected":true}`} {
		s := &alertStub{}
		w := alertRequest(s, "POST", "/listings/l/alerts", body)
		if w.Code != 400 || s.writes != 0 {
			t.Fatalf("%s: %d writes=%d", body, w.Code, s.writes)
		}
	}
	for _, body := range []string{`{"kind":"target","currency":"USD","threshold_minor_units":0}`, `{"kind":"drop","currency":"JPY","drop_basis_points":1250}`, `{"kind":"historical_low","currency":"USD","require_in_stock":false}`} {
		s := &alertStub{}
		w := alertRequest(s, "POST", "/listings/l/alerts", body)
		if w.Code != 201 || len(s.alerts) != 1 {
			t.Fatal(w.Code, w.Body.String())
		}
		var response struct{ Data domain.PriceAlert }
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.Data.ListingID != "l" || response.Data.ID == "" || !response.Data.Enabled {
			t.Fatal(response)
		}
		w = alertRequest(s, "GET", "/listings/l/alerts", "")
		if w.Code != 200 || !strings.Contains(w.Body.String(), response.Data.ID) {
			t.Fatal(w.Body.String())
		}
		w = alertRequest(s, "PATCH", "/listings/l/alerts/"+response.Data.ID, `{"enabled":false}`)
		if w.Code != 200 || s.alerts[0].Enabled {
			t.Fatal(w.Body.String())
		}
		w = alertRequest(s, "GET", "/listings/l/alert-events", "")
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"observation_id":"o"`) {
			t.Fatal(w.Body.String())
		}
	}
	for _, body := range []string{`{}`, `null`, `{"enabled":null}`, `{"enabled":"false"}`} {
		if w := alertRequest(&alertStub{}, "PATCH", "/listings/l/alerts/a", body); w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	if w := alertRequest(&alertStub{}, "PATCH", "/listings/l/alerts/a", `{"enabled":true}`); w.Code != 404 {
		t.Fatal(w.Code)
	}
	for _, tt := range []struct {
		s    *alertStub
		code int
	}{{&alertStub{missing: true}, 404}, {&alertStub{err: errors.New("secret database details")}, 500}, {&alertStub{err: persistence.ErrAlertLimit}, 409}} {
		w := alertRequest(tt.s, "POST", "/listings/l/alerts", `{"kind":"target","currency":"USD","threshold_minor_units":1}`)
		if w.Code != tt.code || strings.Contains(w.Body.String(), "secret") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/listings/l/alerts", "/listings/l/alert-events"} {
		for _, tt := range []struct {
			s    *alertStub
			code int
		}{{&alertStub{missing: true}, 404}, {&alertStub{err: errors.New("secret")}, 500}} {
			w := alertRequest(tt.s, "GET", path, "")
			if w.Code != tt.code || strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w.Code, w.Body.String())
			}
		}
	}
}
