package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type promoStub struct {
	writes                            int
	p                                 domain.Promotion
	observation                       *domain.PriceObservation
	writeErr, readErr, errorPromotion error
}

func (s *promoStub) GetListing(context.Context, string) (domain.Listing, error) {
	return domain.Listing{ID: "l"}, s.readErr
}
func (s *promoStub) CurrentListing(context.Context, string) (persistence.CurrentListing, error) {
	return persistence.CurrentListing{Listing: domain.Listing{ID: "l"}, Observation: s.observation}, s.readErr
}
func (s *promoStub) InsertPromotion(_ context.Context, p domain.Promotion) error {
	s.writes++
	s.p = p
	return s.writeErr
}

func TestPromotionAmountPresence(t *testing.T) {
	for _, kind := range []string{"fixed", "cashback", "membership"} {
		for _, amount := range []struct {
			name, json string
			valid      bool
		}{
			{"missing minor units", `{"currency":"USD"}`, false},
			{"null minor units", `{"minor_units":null,"currency":"USD"}`, false},
			{"explicit zero", `{"minor_units":0,"currency":"USD"}`, true},
			{"null amount", `null`, false},
			{"fractional amount", `{"minor_units":0.5,"currency":"USD"}`, false},
		} {
			t.Run(kind+"/"+amount.name, func(t *testing.T) {
				s := &promoStub{}
				a := promotionAPI{store: s}
				mux := http.NewServeMux()
				mux.HandleFunc("POST /listings/{id}/promotions", a.create)
				requirement := "none"
				if kind == "membership" {
					requirement = "membership"
				}
				body := `{"id":"p","source":"fixture","observed_at":"2026-09-29T00:00:00Z","kind":"` + kind + `","amount":` + amount.json + `,"requirement":"` + requirement + `","stacking":"unknown","terms":"terms"}`
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, httptest.NewRequest("POST", "/listings/l/promotions", strings.NewReader(body)))
				if !amount.valid {
					if w.Code != 400 || s.writes != 0 {
						t.Fatalf("status=%d writes=%d body=%s", w.Code, s.writes, w.Body.String())
					}
					return
				}
				if w.Code != 201 || s.writes != 1 || s.p.Amount == nil || s.p.Amount.MinorUnits != 0 || s.p.Amount.Currency != domain.USD {
					t.Fatalf("zero not preserved: status=%d writes=%d promotion=%+v", w.Code, s.writes, s.p)
				}
				var result struct {
					Data struct {
						Amount struct {
							MinorUnits *int64 `json:"minor_units"`
							Currency   string `json:"currency"`
						} `json:"amount"`
					} `json:"data"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Data.Amount.MinorUnits == nil || *result.Data.Amount.MinorUnits != 0 || result.Data.Amount.Currency != "USD" {
					t.Fatal("response shape/zero changed", w.Body.String())
				}
			})
		}
	}
}
func (s *promoStub) GetPromotion(context.Context, string, string) (domain.Promotion, error) {
	return s.p, s.errorPromotion
}
func (s *promoStub) RelevantPromotions(context.Context, string, time.Time) ([]domain.Promotion, bool, error) {
	return []domain.Promotion{s.p}, false, s.errorPromotion
}
func TestPromotionHTTP(t *testing.T) {
	now := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	o, _ := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: now, Source: "fixture", Stock: domain.StockInStock, OfferPrice: &domain.Money{MinorUnits: 1000, Currency: domain.USD}})
	p := domain.Promotion{ID: "p", ListingID: "l", Source: "fixture", ObservedAt: now, Kind: "fixed", Amount: &domain.Money{MinorUnits: 100, Currency: domain.USD}, Requirement: "none", Stacking: "unknown", Terms: "SAVE"}
	body := `{"id":"p","source":"fixture","observed_at":"2026-09-29T00:00:00Z","kind":"fixed","amount":{"minor_units":100,"currency":"USD"},"requirement":"none","stacking":"unknown","terms":"SAVE"}`
	for _, tc := range []struct {
		name, method, path, body    string
		writeErr, readErr, promoErr error
		want                        int
	}{
		{name: "create", method: "POST", path: "promotions", body: body, want: 201},
		{name: "invalid json", method: "POST", path: "promotions", body: "{", want: 400},
		{name: "invalid domain", method: "POST", path: "promotions", body: "{}", want: 400},
		{name: "duplicate", method: "POST", path: "promotions", body: body, writeErr: &pgconn.PgError{Code: "23505"}, want: 409},
		{name: "missing FK", method: "POST", path: "promotions", body: body, writeErr: &pgconn.PgError{Code: "23503"}, want: 404},
		{name: "write error", method: "POST", path: "promotions", body: body, writeErr: errors.New("secret"), want: 500},
		{name: "list", method: "GET", path: "promotions", want: 200},
		{name: "list error", method: "GET", path: "promotions", promoErr: errors.New("secret"), want: 500},
		{name: "list missing", method: "GET", path: "promotions", readErr: sql.ErrNoRows, want: 404},
		{name: "effective", method: "GET", path: "effective-price?scenario=selected&promotion_id=p", want: 200},
		{name: "missing scenario", method: "GET", path: "effective-price?promotion_id=p", want: 400},
		{name: "unknown parameter", method: "GET", path: "effective-price?scenario=selected&promotion_id=p&membership=yes", want: 400},
		{name: "bad assumption", method: "GET", path: "effective-price?scenario=selected&promotion_id=p&member=true", want: 400},
		{name: "duplicate selection", method: "GET", path: "effective-price?scenario=selected&promotion_id=p&promotion_id=p", want: 400},
		{name: "missing evidence", method: "GET", path: "effective-price?scenario=selected&promotion_id=absent", promoErr: sql.ErrNoRows, want: 404},
		{name: "missing listing", method: "GET", path: "effective-price?scenario=selected&promotion_id=p", readErr: sql.ErrNoRows, want: 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &promoStub{p: p, observation: &o, writeErr: tc.writeErr, readErr: tc.readErr, errorPromotion: tc.promoErr}
			a := promotionAPI{s, func() time.Time { return now }}
			mux := http.NewServeMux()
			mux.HandleFunc("POST /listings/{id}/promotions", a.create)
			mux.HandleFunc("GET /listings/{id}/promotions", a.list)
			mux.HandleFunc("GET /listings/{id}/effective-price", a.effective)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest(tc.method, "/listings/l/"+tc.path, strings.NewReader(tc.body)))
			if w.Code != tc.want || strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w.Code, w.Body.String())
			}
			if tc.name == "effective" {
				var v struct {
					Data struct {
						Status  string         `json:"status"`
						Payable promotionMoney `json:"immediate_payable"`
						Base    promotionMoney `json:"base"`
					}
				}
				if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil || v.Data.Status != "available" || v.Data.Payable.MinorUnits != 900 || v.Data.Base.MinorUnits != 1000 {
					t.Fatal(w.Body.String(), err)
				}
			}
		})
	}
}

func TestEffectiveHTTPWithholdsConditionalTotals(t *testing.T) {
	now := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	o, _ := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: now, Source: "fixture", Stock: domain.StockInStock, OfferPrice: &domain.Money{MinorUnits: 1000, Currency: domain.USD}})
	for _, missing := range []bool{false, true} {
		s := &promoStub{p: domain.Promotion{ID: "p", ListingID: "l", Source: "fixture", ObservedAt: now, Kind: "fixed", Amount: &domain.Money{MinorUnits: 100, Currency: domain.USD}, Requirement: "unknown", Stacking: "unknown", Terms: "eligibility unknown"}, observation: &o}
		if missing {
			s.observation = nil
		}
		a := promotionAPI{s, func() time.Time { return now }}
		mux := http.NewServeMux()
		mux.HandleFunc("GET /listings/{id}/effective-price", a.effective)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/listings/l/effective-price?scenario=selected&promotion_id=p&eligible=yes", nil))
		var result struct {
			Data struct {
				Status  string          `json:"status"`
				Payable *promotionMoney `json:"immediate_payable"`
				Net     *promotionMoney `json:"potential_net"`
			}
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		want := "conditional"
		if missing {
			want = "unavailable"
		}
		if w.Code != 200 || result.Data.Status != want || result.Data.Payable != nil || result.Data.Net != nil {
			t.Fatal(w.Body.String())
		}
	}
}
