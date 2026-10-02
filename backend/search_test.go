package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type searchStub struct {
	calls int
	query string
	err   error
}

func (s *searchStub) SearchCatalog(_ context.Context, q string) (persistence.CatalogSearch, error) {
	s.calls++
	s.query = q
	return persistence.CatalogSearch{Products: []domain.Product{{ID: "constructor", Archived: true}}, Retailers: []domain.Retailer{{ID: "__proto__"}}, Listings: []domain.Listing{{ID: "toString", TrackingDisabled: true}}}, s.err
}
func TestSearchAPI(t *testing.T) {
	for _, query := range []string{"", "   ", "constructor", "https://example.com/?q=%_'\\", "' OR 1=1 --"} {
		s := &searchStub{}
		w := httptest.NewRecorder()
		searchHandler(s)(w, httptest.NewRequest("GET", "/search?q="+url.QueryEscape(query), nil))
		if w.Code != 200 {
			t.Fatal(w)
		}
		blank := strings.TrimSpace(query) == ""
		if (s.calls == 0) != blank {
			t.Fatal("unexpected query count", s.calls)
		}
		var result struct {
			Data struct {
				Products  []productJSON  `json:"products"`
				Retailers []retailerJSON `json:"retailers"`
				Listings  []listingJSON  `json:"listings"`
			}
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if !blank && (s.query != query || !result.Data.Products[0].Archived || result.Data.Listings[0].TrackingEnabled) {
			t.Fatal(result, s)
		}
	}
	for _, query := range []string{"q=a&q=b", "q=%00", "q=" + strings.Repeat("x", 201), "q=%ZZ"} {
		s := &searchStub{}
		w := httptest.NewRecorder()
		searchHandler(s)(w, httptest.NewRequest("GET", "/search?"+query, nil))
		if w.Code != 400 || s.calls != 0 {
			t.Fatal(w, s)
		}
	}
	s := &searchStub{err: errors.New("secret")}
	w := httptest.NewRecorder()
	searchHandler(s)(w, httptest.NewRequest("GET", "/search?q=x", nil))
	if w.Code != 500 || strings.Contains(w.Body.String(), "secret") {
		t.Fatal(w)
	}
}
