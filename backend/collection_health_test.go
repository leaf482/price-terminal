package main

import (
	"context"
	"errors"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http/httptest"
	"strings"
	"testing"
)

type healthOverviewStub struct {
	calls int
	err   error
}

func (s *healthOverviewStub) CollectionHealth(context.Context) (persistence.CollectionHealth, error) {
	s.calls++
	return persistence.CollectionHealth{Listings: []persistence.CollectionHealthListing{{ListingID: "constructor"}, {ListingID: "__proto__"}}}, s.err
}
func TestCollectionHealthAPI(t *testing.T) {
	for _, fail := range []bool{false, true} {
		s := &healthOverviewStub{}
		if fail {
			s.err = errors.New("secret")
		}
		w := httptest.NewRecorder()
		collectionHealthHandler(s)(w, httptest.NewRequest("GET", "/collection/overview", nil))
		if s.calls != 1 {
			t.Fatal("expected one bounded store call")
		}
		if fail {
			if w.Code != 500 || strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w.Body.String())
			}
		} else if w.Code != 200 || !strings.Contains(w.Body.String(), "__proto__") {
			t.Fatal(w.Body.String())
		}
	}
}
