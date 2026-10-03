package main

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/leaf482/price-terminal/backend/domain"
)

// Characterize differences that a common decoder must not flatten. Persistence
// callbacks are absent so every rejected request must stop before a write.
func TestCreateJSONCompatibility(t *testing.T) {
	for _, tt := range []struct{ name, body, productMessage string }{
		{"malformed", `{`, "expected a product JSON object"},
		{"empty", "", "expected a product JSON object"},
		{"null", `null`, "expected a product JSON object"},
		{"null with trailing data", `null {}`, "expected a product JSON object"},
		{"unknown field", `{"id":"p","unknown":0}`, "expected a product JSON object"},
		{"second value", `{"id":"p"} {}`, "expected exactly one JSON object"},
		{"trailing garbage", `{"id":"p"} garbage`, "expected exactly one JSON object"},
		{"oversized object", `{"id":"p","name":"` + strings.Repeat("x", 65536) + `"}`, "expected a product JSON object"},
		{"oversized trailing whitespace", `{"id":"p"}` + strings.Repeat(" ", 65536), "expected exactly one JSON object"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, endpoint := range []struct{ path, message string }{{"/products", tt.productMessage}, {"/retailers", "expected a retailer JSON object"}} {
				w := httptest.NewRecorder()
				newHandler(nil, productStub{}, catalogStub{}).ServeHTTP(w, httptest.NewRequest("POST", endpoint.path, strings.NewReader(tt.body)))
				want := `{"error":{"code":"invalid_request","message":"` + endpoint.message + `"}}` + "\n"
				if w.Code != 400 || w.Body.String() != want || w.Header().Get("Content-Type") != "application/json" {
					t.Fatalf("%s: %d %s", endpoint.path, w.Code, w.Body.String())
				}
			}
		})
	}
}

func TestJSONCompatibilityKeepsContentTypeAndDuplicateKeyBehavior(t *testing.T) {
	called := false
	s := productStub{insert: func(_ context.Context, p domain.Product) error {
		called = true
		if p.ID != "last" {
			t.Fatalf("duplicate key semantics changed: %q", p.ID)
		}
		return nil
	}}
	r := httptest.NewRequest("POST", "/products", strings.NewReader(`{"id":"first","id":"last"} `))
	r.Header.Set("Content-Type", "text/plain") // Existing API does not require a JSON media type.
	w := httptest.NewRecorder()
	newHandler(nil, s, nil).ServeHTTP(w, r)
	if w.Code != 201 || !called {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestListQueryCompatibility(t *testing.T) {
	for _, query := range []string{"limit=%zz", "other=%zz", "other=a;b"} {
		products := productStub{list: func(_ context.Context, limit int) ([]domain.Product, error) {
			if limit != 50 {
				t.Fatalf("limit=%d", limit)
			}
			return nil, nil
		}}
		w := httptest.NewRecorder()
		newHandler(nil, products, nil).ServeHTTP(w, httptest.NewRequest("GET", "/products?"+query, nil))
		if w.Code != 200 || w.Body.String() != `{"data":[]}`+"\n" {
			t.Fatalf("Product query %q: %d %s", query, w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		newHandler(nil, nil, catalogStub{}).ServeHTTP(w, httptest.NewRequest("GET", "/retailers?"+query, nil))
		if w.Code != 400 || w.Body.String() != `{"error":{"code":"invalid_limit","message":"limit must be an integer from 1 to 100"}}`+"\n" {
			t.Fatalf("Retailer query %q: %d %s", query, w.Code, w.Body.String())
		}
	}
}
