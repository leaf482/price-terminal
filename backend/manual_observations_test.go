package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
)

type manualStore struct {
	observation                   domain.PriceObservation
	writes, evaluations           int
	lookupErr, writeErr, alertErr error
}

func (s *manualStore) GetListing(context.Context, string) (domain.Listing, error) {
	return domain.Listing{ID: "l", ProductID: "p", RetailerID: "r", URL: "https://example.com/l"}, s.lookupErr
}
func (s *manualStore) InsertPriceObservation(_ context.Context, id string, o domain.PriceObservation) error {
	if id != "entry" {
		return errors.New("unexpected ID")
	}
	s.writes++
	s.observation = o
	return s.writeErr
}
func (s *manualStore) EvaluateAlerts(context.Context, string) error {
	s.evaluations++
	return s.alertErr
}

func TestManualObservation(t *testing.T) {
	money := func(n any, c string) map[string]any { return map[string]any{"minor_units": n, "currency": c} }
	for _, tc := range []struct {
		name   string
		change func(map[string]any)
		status int
	}{
		{"stock only", func(b map[string]any) {}, 201},
		{"full USD", func(b map[string]any) {
			b["msrp"] = money(150, "USD")
			b["msrp_source"] = "manufacturer claim"
			b["retailer_list_price"] = money(120, "USD")
			b["sale_price"] = money(100, "USD")
			b["offer_price"] = money(0, "USD")
		}, 201},
		{"JPY zero", func(b map[string]any) { b["offer_price"] = money(0, "JPY") }, 201},
		{"mixed", func(b map[string]any) { b["sale_price"] = money(1, "USD"); b["offer_price"] = money(1, "JPY") }, 400},
		{"missing units", func(b map[string]any) { b["offer_price"] = map[string]any{"currency": "USD"} }, 400},
		{"null units", func(b map[string]any) { b["offer_price"] = money(nil, "USD") }, 400},
		{"negative", func(b map[string]any) { b["offer_price"] = money(-1, "USD") }, 400},
		{"unsupported currency", func(b map[string]any) { b["offer_price"] = money(1, "EUR") }, 400},
		{"MSRP evidence missing", func(b map[string]any) { b["msrp"] = money(1, "USD") }, 400},
		{"evidence without MSRP", func(b map[string]any) { b["msrp_source"] = "claim" }, 400},
		{"invalid stock", func(b map[string]any) { b["stock"] = "yes" }, 400},
		{"missing time", func(b map[string]any) { delete(b, "observed_at") }, 400},
		{"missing source", func(b map[string]any) { b["source"] = " " }, 400},
		{"missing ID", func(b map[string]any) { delete(b, "id") }, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{"id": "entry", "observed_at": "2026-09-30T12:00:00+02:00", "source": "receipt", "stock": "unknown"}
			tc.change(body)
			encoded, _ := json.Marshal(body)
			store := &manualStore{}
			req := httptest.NewRequest("POST", "/listings/l/observations", strings.NewReader(string(encoded)))
			req.SetPathValue("id", "l")
			w := httptest.NewRecorder()
			manualObservationHandler(store)(w, req)
			if w.Code != tc.status {
				t.Fatalf("%d %s", w.Code, w.Body)
			}
			if tc.status != 201 {
				if store.writes != 0 || store.evaluations != 0 {
					t.Fatal("invalid input persisted")
				}
				return
			}
			if store.writes != 1 || store.evaluations != 1 || store.observation.Source() != "manual: receipt" || store.observation.ObservedAt().Hour() != 10 {
				t.Fatal("shared persistence/evaluation or provenance/time lost")
			}
			if tc.name == "stock only" {
				if _, ok := store.observation.Currency(); ok {
					t.Fatal("invented currency")
				}
				if _, ok := store.observation.OfferPrice(); ok {
					t.Fatal("invented price")
				}
			}
			if tc.name == "full USD" || tc.name == "JPY zero" {
				if p, ok := store.observation.OfferPrice(); !ok || p.MinorUnits != 0 {
					t.Fatal("zero lost")
				}
			}
			if tc.name == "full USD" {
				if p, ok := store.observation.MSRP(); !ok || p.MinorUnits != 150 || store.observation.MSRPSource() != "manufacturer claim" {
					t.Fatal("MSRP lost")
				}
				if p, _ := store.observation.RetailerListPrice(); p.MinorUnits != 120 {
					t.Fatal("list lost")
				}
				if p, _ := store.observation.SalePrice(); p.MinorUnits != 100 {
					t.Fatal("sale lost")
				}
			}
		})
	}
}
func TestManualErrors(t *testing.T) {
	valid := `{"id":"entry","source":"receipt","stock":"unknown","observed_at":"2026-09-30T00:00:00Z"}`
	for _, tc := range []struct {
		name, body string
		store      manualStore
		status     int
	}{
		{"malformed", "{", manualStore{}, 400}, {"null", "null", manualStore{}, 400},
		{"unknown field", `{"bad":true}`, manualStore{}, 400},
		{"missing listing", valid, manualStore{lookupErr: sql.ErrNoRows}, 404},
		{"database error", valid, manualStore{writeErr: errors.New("secret")}, 500},
		{"duplicate", valid, manualStore{writeErr: &pgconn.PgError{Code: "23505"}}, 409},
		{"alert failure retains observation", valid, manualStore{alertErr: errors.New("secret")}, 201},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			r.SetPathValue("id", "l")
			w := httptest.NewRecorder()
			manualObservationHandler(&tc.store)(w, r)
			if w.Code != tc.status || strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w.Code, w.Body)
			}
			if tc.status == 400 || tc.status == 404 {
				if tc.store.writes != 0 {
					t.Fatal("unexpected write")
				}
			}
			if tc.store.writeErr != nil && tc.store.evaluations != 0 {
				t.Fatal("evaluated failed write")
			}
		})
	}
}

func TestManualTimestampValidationAndPrecision(t *testing.T) {
	for _, tc := range []struct {
		input   string
		wantUTC string
	}{
		{"2026-02-30T12:00:00Z", ""},
		{"", ""},
		{"2026-01-02T03:04:05.123456789Z", "2026-01-02T03:04:05.123456789Z"},
		{"2026-01-02T03:04:05.123456789+05:30", "2026-01-01T21:34:05.123456789Z"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{"id": "entry", "source": "receipt", "stock": "unknown", "observed_at": tc.input})
			r := httptest.NewRequest("POST", "/listings/l/observations", strings.NewReader(string(body)))
			r.SetPathValue("id", "l")
			w := httptest.NewRecorder()
			store := &manualStore{}
			manualObservationHandler(store)(w, r)
			if tc.wantUTC == "" {
				if w.Code != 400 || store.writes != 0 || store.evaluations != 0 {
					t.Fatalf("invalid calendar/time accepted: %d writes=%d", w.Code, store.writes)
				}
				return
			}
			if w.Code != 201 || store.writes != 1 {
				t.Fatalf("%d %s", w.Code, w.Body)
			}
			if got := store.observation.ObservedAt().Format(time.RFC3339Nano); got != tc.wantUTC {
				t.Fatalf("persisted time %s; want %s", got, tc.wantUTC)
			}
		})
	}
}
