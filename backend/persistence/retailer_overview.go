package persistence

import (
	"context"
	"github.com/leaf482/price-terminal/backend/domain"
)

type RetailerListing struct {
	Product domain.Product
	Current CurrentListing
}
type retailerScanner struct {
	row     interface{ Scan(...any) error }
	product *domain.Product
}

func (s retailerScanner) Scan(dest ...any) error {
	p := s.product
	return s.row.Scan(append([]any{&p.ID, &p.Name, &p.Brand, &p.Model, &p.Archived}, dest...)...)
}

// A single bounded statement joins catalog metadata to coherent latest valid
// observations. The extra row reports truncation, never an unbounded overview.
func (s *Store) RetailerOverview(ctx context.Context, id string) ([]RetailerListing, bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT p.id,p.name,p.brand,p.model,p.archived,snapshot.* FROM (`+currentSelect+`WHERE l.retailer_id=$1 ORDER BY l.id COLLATE "C" LIMIT $2) snapshot JOIN products p ON p.id=snapshot.product_id ORDER BY snapshot.id COLLATE "C"`, id, MaxCurrentListings+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	result := make([]RetailerListing, 0)
	for rows.Next() {
		var item RetailerListing
		current, err := scanCurrent(retailerScanner{rows, &item.Product})
		if err != nil {
			return nil, false, err
		}
		if err = item.Product.Validate(); err != nil {
			return nil, false, err
		}
		item.Current = current
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	more := len(result) > MaxCurrentListings
	if more {
		result = result[:MaxCurrentListings]
	}
	return result, more, nil
}
