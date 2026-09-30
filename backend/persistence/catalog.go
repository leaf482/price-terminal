// Package persistence stores catalog and observation domain values using database/sql. The
// caller owns the database pool and supplies request deadlines via context.
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/leaf482/price-terminal/backend/domain"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) InsertProduct(ctx context.Context, product domain.Product) error {
	if err := product.Validate(); err != nil {
		return fmt.Errorf("insert product: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO products (id, name, brand, model) VALUES ($1, $2, $3, $4)`,
		product.ID, product.Name, product.Brand, product.Model)
	if err != nil {
		return fmt.Errorf("insert product: %w", err)
	}
	return nil
}

func (s *Store) GetProduct(ctx context.Context, id string) (domain.Product, error) {
	var product domain.Product
	err := s.db.QueryRowContext(ctx, `SELECT id, name, brand, model FROM products WHERE id = $1`, id).
		Scan(&product.ID, &product.Name, &product.Brand, &product.Model)
	if err != nil {
		return domain.Product{}, fmt.Errorf("get product: %w", err)
	}
	if err := product.Validate(); err != nil {
		return domain.Product{}, fmt.Errorf("get product: %w", err)
	}
	return product, nil
}

const MaxProductListLimit = 100

// ListProducts returns at most limit products in stable primary-key order.
// Callers must explicitly select a bound; zero never means an unbounded query.
func (s *Store) ListProducts(ctx context.Context, limit int) ([]domain.Product, error) {
	if limit < 1 || limit > MaxProductListLimit {
		return nil, fmt.Errorf("list products: limit must be between 1 and %d", MaxProductListLimit)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, brand, model FROM products ORDER BY id LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()
	products := make([]domain.Product, 0)
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Brand, &p.Model); err != nil {
			return nil, fmt.Errorf("list products: %w", err)
		}
		if err := p.Validate(); err != nil {
			return nil, fmt.Errorf("list products: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	return products, nil
}

func (s *Store) InsertRetailer(ctx context.Context, retailer domain.Retailer) error {
	if err := retailer.Validate(); err != nil {
		return fmt.Errorf("insert retailer: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO retailers (id, name) VALUES ($1, $2)`, retailer.ID, retailer.Name)
	if err != nil {
		return fmt.Errorf("insert retailer: %w", err)
	}
	return nil
}

func (s *Store) GetRetailer(ctx context.Context, id string) (domain.Retailer, error) {
	var retailer domain.Retailer
	err := s.db.QueryRowContext(ctx, `SELECT id, name FROM retailers WHERE id = $1`, id).
		Scan(&retailer.ID, &retailer.Name)
	if err != nil {
		return domain.Retailer{}, fmt.Errorf("get retailer: %w", err)
	}
	if err := retailer.Validate(); err != nil {
		return domain.Retailer{}, fmt.Errorf("get retailer: %w", err)
	}
	return retailer, nil
}

// InsertListing is one atomic INSERT. It neither creates related records nor
// updates an existing listing. PostgreSQL enforces references and exact source
// identity; wrapped driver errors retain their SQLSTATE via errors.As.
func (s *Store) InsertListing(ctx context.Context, listing domain.Listing) error {
	if err := listing.Validate(); err != nil {
		return fmt.Errorf("insert listing: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO listings (id, product_id, retailer_id, url, retailer_product_id, tracking_enabled)
		VALUES ($1, $2, $3, $4, $5, $6)`, listing.ID, listing.ProductID, listing.RetailerID, listing.URL, listing.RetailerProductID, listing.TrackingEnabled())
	if err != nil {
		return fmt.Errorf("insert listing: %w", err)
	}
	return nil
}

func (s *Store) GetListing(ctx context.Context, id string) (domain.Listing, error) {
	var listing domain.Listing
	err := s.db.QueryRowContext(ctx, `SELECT id, product_id, retailer_id, url, retailer_product_id, NOT tracking_enabled FROM listings WHERE id = $1`, id).
		Scan(&listing.ID, &listing.ProductID, &listing.RetailerID, &listing.URL, &listing.RetailerProductID, &listing.TrackingDisabled)
	if err != nil {
		return domain.Listing{}, fmt.Errorf("get listing: %w", err)
	}
	if err := listing.Validate(); err != nil {
		return domain.Listing{}, fmt.Errorf("get listing: %w", err)
	}
	return listing, nil
}

func (s *Store) SetListingTracking(ctx context.Context, id string, enabled bool) error {
	result, err := s.db.ExecContext(ctx, `UPDATE listings SET tracking_enabled=$2 WHERE id=$1`, id, enabled)
	if err != nil {
		return fmt.Errorf("set tracking: %w", err)
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
