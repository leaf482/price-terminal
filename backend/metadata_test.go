package main

import (
	"context"
	"database/sql"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type metadataStub struct {
	calls int
	p     domain.Product
	r     domain.Retailer
	err   error
}

func (s *metadataStub) UpdateProductMetadata(_ context.Context, p domain.Product) (domain.Product, error) {
	s.calls++
	s.p = p
	return p, s.err
}
func (s *metadataStub) UpdateRetailerMetadata(_ context.Context, r domain.Retailer) (domain.Retailer, error) {
	s.calls++
	s.r = r
	return r, s.err
}
func TestMetadataUpdates(t *testing.T) {
	for _, tt := range []struct {
		path, body    string
		err           error
		status, calls int
	}{
		{"/products/p", `{"name":"New","brand":"Maker","model":""}`, nil, 200, 1},
		{"/retailers/r", `{"name":""}`, nil, 200, 1},
		{"/products/p", `{"name":"New"}`, nil, 400, 0},
		{"/products/p", `{"name":null,"brand":"","model":""}`, nil, 400, 0},
		{"/products/p", `{"id":"other","name":"","brand":"","model":""}`, nil, 400, 0},
		{"/retailers/r", `{"id":"other","name":"New"}`, nil, 400, 0},
		{"/retailers/r", `{"name":2}`, nil, 400, 0},
		{"/retailers/r", `null`, nil, 400, 0},
		{"/retailers/r", `{`, nil, 400, 0},
		{"/products/%20", `{"name":"","brand":"","model":""}`, nil, 400, 0},
		{"/products/p", `{"name":"","brand":"","model":""}`, sql.ErrNoRows, 404, 1},
		{"/retailers/r", `{"name":"New"}`, sql.ErrNoRows, 404, 1},
		{"/retailers/r", `{"name":"New"}`, errors.New("secret"), 500, 1},
	} {
		t.Run(tt.path+tt.body, func(t *testing.T) {
			s := &metadataStub{err: tt.err}
			mux := http.NewServeMux()
			registerMetadataRoutes(mux, s)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("PUT", tt.path, strings.NewReader(tt.body)))
			if w.Code != tt.status || s.calls != tt.calls || strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w, s)
			}
			if w.Code == 200 && s.p.ID == "p" && (s.p.Name != "New" || s.p.Brand != "Maker" || s.p.Model != "") {
				t.Fatal(s.p)
			}
		})
	}
	// No source mutation route is registered alongside existing catalog routes.
	mux := http.NewServeMux()
	registerCatalogRoutes(mux, catalogStub{})
	registerMetadataRoutes(mux, &metadataStub{})
	for _, method := range []string{"PUT", "PATCH"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, "/listings/l", strings.NewReader(`{"url":"https://other.example"}`)))
		if w.Code != 405 {
			t.Fatal(w)
		}
	}
}
