package persistence

import (
	"context"
	"fmt"
	"strings"

	"github.com/leaf482/price-terminal/backend/domain"
)

const MaxCatalogListLimit = 100

// ListRetailers requires an explicit bound and returns stable primary-key order.
func (s *Store) ListRetailers(ctx context.Context, limit int) ([]domain.Retailer, error) {
	if limit < 1 || limit > MaxCatalogListLimit {
		return nil, fmt.Errorf("list retailers: limit must be between 1 and %d", MaxCatalogListLimit)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, name FROM retailers ORDER BY id LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list retailers: %w", err)
	}
	defer rows.Close()
	retailers := make([]domain.Retailer, 0)
	for rows.Next() {
		var retailer domain.Retailer
		if err := rows.Scan(&retailer.ID, &retailer.Name); err != nil {
			return nil, fmt.Errorf("list retailers: %w", err)
		}
		if err := retailer.Validate(); err != nil {
			return nil, fmt.Errorf("list retailers: %w", err)
		}
		retailers = append(retailers, retailer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list retailers: %w", err)
	}
	return retailers, nil
}

// ListListingsByProduct preserves stored source strings exactly. Missing products
// and products without listings both return an empty slice; HTTP checks the parent.
func (s *Store) ListListingsByProduct(ctx context.Context, productID string, limit int) ([]domain.Listing, error) {
	if strings.TrimSpace(productID) == "" {
		return nil, fmt.Errorf("list listings: product ID is required")
	}
	if limit < 1 || limit > MaxCatalogListLimit {
		return nil, fmt.Errorf("list listings: limit must be between 1 and %d", MaxCatalogListLimit)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, product_id, retailer_id, url, retailer_product_id, NOT tracking_enabled
		FROM listings WHERE product_id = $1 ORDER BY id LIMIT $2`, productID, limit)
	if err != nil {
		return nil, fmt.Errorf("list listings: %w", err)
	}
	defer rows.Close()
	listings := make([]domain.Listing, 0)
	for rows.Next() {
		var listing domain.Listing
		if err := rows.Scan(&listing.ID, &listing.ProductID, &listing.RetailerID, &listing.URL, &listing.RetailerProductID, &listing.TrackingDisabled); err != nil {
			return nil, fmt.Errorf("list listings: %w", err)
		}
		if err := listing.Validate(); err != nil {
			return nil, fmt.Errorf("list listings: %w", err)
		}
		listings = append(listings, listing)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list listings: %w", err)
	}
	return listings, nil
}
