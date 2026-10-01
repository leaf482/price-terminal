package persistence

import (
	"context"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
	"strings"
)

const MaxImportRows = 500

// ImportObservations is an explicit manual operation, allowed while tracking is
// disabled. It deliberately bypasses post-write alert evaluation (no backfill).
func (s *Store) ImportObservations(ctx context.Context, listingID, importID string, observations []domain.PriceObservation) error {
	if strings.TrimSpace(importID) == "" || strings.TrimSpace(listingID) == "" || len(observations) == 0 || len(observations) > MaxImportRows {
		return fmt.Errorf("invalid observation import")
	}
	for _, o := range observations {
		if err := o.Validate(); err != nil {
			return err
		}
		if o.ListingID() != listingID {
			return fmt.Errorf("import listing mismatch")
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, o := range observations {
		if err := insertPriceObservation(ctx, tx, fmt.Sprintf("csv:%s:%d", importID, i+1), o); err != nil {
			return fmt.Errorf("import row %d: %w", i+2, err)
		}
	}
	return tx.Commit()
}
