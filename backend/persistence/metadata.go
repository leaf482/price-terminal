package persistence

import (
	"context"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
)

// UpdateProductMetadata replaces descriptive fields only; identity is the WHERE
// key, never an assignment. Empty descriptions retain their domain meaning.
func (s *Store) UpdateProductMetadata(ctx context.Context, p domain.Product) (domain.Product, error) {
	if err := p.Validate(); err != nil {
		return domain.Product{}, err
	}
	var saved domain.Product
	err := s.db.QueryRowContext(ctx, `UPDATE products SET name=$2,brand=$3,model=$4 WHERE id=$1 RETURNING id,name,brand,model`, p.ID, p.Name, p.Brand, p.Model).Scan(&saved.ID, &saved.Name, &saved.Brand, &saved.Model)
	if err != nil {
		return domain.Product{}, fmt.Errorf("update product metadata: %w", err)
	}
	return saved, nil
}

func (s *Store) UpdateRetailerMetadata(ctx context.Context, r domain.Retailer) (domain.Retailer, error) {
	if err := r.Validate(); err != nil {
		return domain.Retailer{}, err
	}
	var saved domain.Retailer
	err := s.db.QueryRowContext(ctx, `UPDATE retailers SET name=$2 WHERE id=$1 RETURNING id,name`, r.ID, r.Name).Scan(&saved.ID, &saved.Name)
	if err != nil {
		return domain.Retailer{}, fmt.Errorf("update retailer metadata: %w", err)
	}
	return saved, nil
}
