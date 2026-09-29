//go:build integration

package persistence_test

import (
	"context"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"testing"
	"time"
)

func TestProductListIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	store := persistence.New(freshDatabase(t, ctx))
	got, err := store.ListProducts(ctx, 1)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty list=%v, %v", got, err)
	}
	for i := 104; i >= 0; i-- {
		p := domain.Product{ID: fmt.Sprintf("p%03d", i), Name: "Name", Brand: "Brand", Model: "Model"}
		if err := store.InsertProduct(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	for _, limit := range []int{1, 50, 100} {
		got, err := store.ListProducts(ctx, limit)
		if err != nil || len(got) != limit {
			t.Fatalf("limit=%d, count=%d, err=%v", limit, len(got), err)
		}
		for i, p := range got {
			want := domain.Product{ID: fmt.Sprintf("p%03d", i), Name: "Name", Brand: "Brand", Model: "Model"}
			if p != want {
				t.Fatalf("row %d=%+v, want %+v", i, p, want)
			}
		}
	}
}
