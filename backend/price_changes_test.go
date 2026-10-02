package main

import (
	"context"
	"errors"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http/httptest"
	"strings"
	"testing"
)

type changesStub struct {
	calls int
	err   error
}

func (s *changesStub) RecentPriceChanges(context.Context) (persistence.PriceChanges, error) {
	s.calls++
	return persistence.PriceChanges{Changes: []persistence.PriceChange{}}, s.err
}
func TestPriceChangesAPI(t *testing.T) {
	for _, fail := range []bool{false, true} {
		s := &changesStub{}
		if fail {
			s.err = errors.New("secret")
		}
		w := httptest.NewRecorder()
		priceChangesHandler(s)(w, httptest.NewRequest("GET", "/price-changes", nil))
		if s.calls != 1 || strings.Contains(w.Body.String(), "secret") {
			t.Fatal(w.Body.String())
		}
		if fail {
			if w.Code != 500 {
				t.Fatal(w.Code)
			}
		} else if w.Code != 200 || !strings.Contains(w.Body.String(), `"changes":[]`) {
			t.Fatal(w.Body.String())
		}
	}
}
