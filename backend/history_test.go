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
	"testing"
	"time"
)

type historyStub struct {
	rows           persistence.History
	err, lookupErr error
	from           *time.Time
	to             time.Time
}

func (s *historyStub) GetListing(context.Context, string) (domain.Listing, error) {
	return domain.Listing{ID: "l"}, s.lookupErr
}
func (s *historyStub) PriceHistory(_ context.Context, _ string, from *time.Time, to time.Time) (persistence.History, error) {
	s.from = from
	s.to = to
	return s.rows, s.err
}
func TestHistoryRangesAndHTTP(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 123456789, time.UTC)
	for name, days := range map[string]int{"1D": 1, "1W": 7, "1M": 30, "3M": 90, "1Y": 365, "ALL": 0, "": 30} {
		t.Run(name, func(t *testing.T) {
			s := &historyStub{}
			mux := http.NewServeMux()
			mux.HandleFunc("GET /listings/{id}/history", historyHandler(s, func() time.Time { return now }))
			path := "/listings/l/history"
			if name != "" {
				path += "?range=" + name
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
			if w.Code != 200 {
				t.Fatal(w.Body.String())
			}
			if !s.to.Equal(now) || (days == 0 && s.from != nil) || (days > 0 && (s.from == nil || !s.from.Equal(now.Add(-time.Duration(days)*24*time.Hour)))) {
				t.Fatal("range bounds", s.from)
			}
			var body struct {
				Data struct {
					Observations []observationJSON `json:"observations"`
				}
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Data.Observations == nil {
				t.Fatal("empty must be array", err)
			}
		})
	}
	for _, tc := range []struct {
		query string
		s     historyStub
		code  int
	}{{"?range=bad", historyStub{}, 400}, {"?range=", historyStub{}, 400}, {"?range=ALL&range=1D", historyStub{}, 400}, {"", historyStub{lookupErr: sql.ErrNoRows}, 404}, {"", historyStub{err: errors.New("secret")}, 500}} {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /listings/{id}/history", historyHandler(&tc.s, func() time.Time { return now }))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/listings/l/history"+tc.query, nil))
		if w.Code != tc.code {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
func TestHistoryMetadata(t *testing.T) {
	makeRow := func(amount *int64, c domain.Currency, sec int) domain.PriceObservation {
		var money *domain.Money
		if amount != nil {
			money = &domain.Money{MinorUnits: *amount, Currency: c}
		}
		o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: time.Unix(int64(sec+1), 0), Source: "fixture", Stock: domain.StockUnknown, SalePrice: money})
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	for _, tc := range []struct {
		name      string
		rows      []domain.PriceObservation
		truncated bool
		low       *int64
		change    string
	}{
		{"empty", nil, false, nil, ""},
		{"stock", []domain.PriceObservation{makeRow(nil, "", 0)}, false, nil, ""},
		{"single zero", []domain.PriceObservation{makeRow(intPrice(0), domain.USD, 0)}, false, intPrice(0), ""},
		{"change", []domain.PriceObservation{makeRow(intPrice(200), domain.USD, 0), makeRow(intPrice(100), domain.USD, 1)}, false, intPrice(100), "-50.00"},
		{"JPY", []domain.PriceObservation{makeRow(intPrice(3), domain.JPY, 0), makeRow(intPrice(4), domain.JPY, 1)}, false, intPrice(3), "33.33"},
		{"missing endpoint", []domain.PriceObservation{makeRow(nil, "", 0), makeRow(intPrice(5), domain.USD, 1)}, false, intPrice(5), ""},
		{"zero start", []domain.PriceObservation{makeRow(intPrice(0), domain.USD, 0), makeRow(intPrice(5), domain.USD, 1)}, false, intPrice(0), ""},
		{"mixed", []domain.PriceObservation{makeRow(intPrice(5), domain.USD, 0), makeRow(intPrice(5), domain.JPY, 1)}, false, nil, ""},
		{"truncated", []domain.PriceObservation{makeRow(intPrice(5), domain.USD, 0)}, true, nil, ""},
		{"same time", []domain.PriceObservation{makeRow(intPrice(5), domain.USD, 0), makeRow(intPrice(6), domain.USD, 0)}, false, intPrice(5), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			low, change := historyMetadata(tc.rows, tc.truncated)
			if (low == nil) != (tc.low == nil) || (low != nil && low.MinorUnits != *tc.low) {
				t.Fatal(low)
			}
			if tc.change == "" {
				if change != nil {
					t.Fatal(*change)
				}
			} else if change == nil || *change != tc.change {
				t.Fatal(change)
			}
		})
	}
}

func TestHistoryObservationFields(t *testing.T) {
	at := time.Date(2026, 9, 29, 0, 0, 0, 123456789, time.UTC)
	zero := domain.Money{Currency: domain.USD, MinorUnits: 0}
	price := domain.Money{Currency: domain.USD, MinorUnits: 500}
	full, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: at, Source: "fixture", Stock: domain.StockInStock, MSRP: &price, MSRPSource: "manufacturer", RetailerListPrice: &price, SalePrice: &price, OfferPrice: &zero})
	if err != nil {
		t.Fatal(err)
	}
	stock, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: at.Add(time.Second), Source: "fixture", Stock: domain.StockUnknown})
	if err != nil {
		t.Fatal(err)
	}
	s := &historyStub{rows: persistence.History{Observations: []domain.PriceObservation{full, stock}}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /listings/{id}/history", historyHandler(s, func() time.Time { return at.Add(time.Minute) }))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/listings/l/history?range=ALL", nil))
	var result struct {
		Data struct {
			Observations []observationJSON `json:"observations"`
		}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || len(result.Data.Observations) != 2 {
		t.Fatal(w.Body.String())
	}
	a, b := result.Data.Observations[0], result.Data.Observations[1]
	if a.MSRP == nil || *a.MSRP != 500 || a.MSRPSource != "manufacturer" || a.RetailerListPrice == nil || a.SalePrice == nil || a.OfferPrice == nil || *a.OfferPrice != 0 || a.Currency != domain.USD || !a.ObservedAt.Equal(at) {
		t.Fatal("lost observed fields", a)
	}
	if b.Currency != "" || b.OfferPrice != nil || b.Stock != domain.StockUnknown {
		t.Fatal("invented stock-only fields", b)
	}
	if p, ok := observedBasis(full); !ok || p != zero {
		t.Fatal("offer must precede sale")
	}
}
