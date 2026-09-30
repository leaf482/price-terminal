package persistence

import (
	"context"
	"testing"
)

func TestCurrentProductsInput(t *testing.T) {
	// A nil DB proves that empty/invalid input never performs database work.
	s := New(nil)
	got, err := s.CurrentProducts(context.Background(), nil)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatal(got, err)
	}
	for _, ids := range [][]string{{""}, {" "}, make([]string, MaxDashboardProducts+1)} {
		if _, err := s.CurrentProducts(context.Background(), ids); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
}
