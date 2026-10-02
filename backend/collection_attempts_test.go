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

type attemptStub struct {
	lookupErr, readErr error
	calls              int
}

func (s *attemptStub) GetListing(context.Context, string) (domain.Listing, error) {
	return domain.Listing{ID: "constructor"}, s.lookupErr
}
func (s *attemptStub) ListCollectionAttempts(context.Context, string) ([]domain.CollectionAttempt, error) {
	s.calls++
	return []domain.CollectionAttempt{}, s.readErr
}
func TestAttemptAPI(t *testing.T) {
	for _, tc := range []struct {
		lookup, read  error
		status, calls int
	}{{nil, nil, 200, 1}, {sql.ErrNoRows, nil, 404, 0}, {nil, errors.New("secret"), 500, 1}} {
		s := &attemptStub{lookupErr: tc.lookup, readErr: tc.read}
		mux := http.NewServeMux()
		registerAttemptRoutes(mux, s)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/listings/constructor/collection-attempts", nil))
		if w.Code != tc.status || s.calls != tc.calls || strings.Contains(w.Body.String(), "secret") {
			t.Fatal(w.Code, w.Body.String(), s.calls)
		}
		if w.Code == 200 && !strings.Contains(w.Body.String(), `"data":[]`) {
			t.Fatal(w.Body.String())
		}
	}
}
