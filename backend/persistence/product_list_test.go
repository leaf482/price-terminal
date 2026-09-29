package persistence_test

import (
	"context"
	"github.com/leaf482/price-terminal/backend/persistence"
	"testing"
)

func TestProductListRejectsInvalidBoundsBeforeSQL(t *testing.T) {
	for _, limit := range []int{-1, 0, persistence.MaxProductListLimit + 1} {
		if _, err := persistence.New(nil).ListProducts(context.Background(), limit); err == nil {
			t.Fatalf("accepted limit %d", limit)
		}
	}
}
