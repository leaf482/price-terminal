package persistence_test

import (
	"context"
	"github.com/leaf482/price-terminal/backend/persistence"
	"testing"
)

func TestBlankSearchWithoutDatabase(t *testing.T) {
	r, err := persistence.New(nil).SearchCatalog(context.Background(), " \t ")
	if err != nil || len(r.Products)+len(r.Retailers)+len(r.Listings) != 0 {
		t.Fatal(r, err)
	}
}
