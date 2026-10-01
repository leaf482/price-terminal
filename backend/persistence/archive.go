package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
)

// Product visibility is independent of Listing collection controls.
func (s *Store) SetProductArchived(ctx context.Context, id string, archived bool) error {
	if err := (domain.Product{ID: id}).Validate(); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE products SET archived=$2 WHERE id=$1`, id, archived)
	if err != nil {
		return fmt.Errorf("archive product: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (s *Store) ListDashboardProducts(ctx context.Context, limit int, includeArchived bool) ([]domain.Product, error) {
	return s.listProducts(ctx, limit, includeArchived)
}
