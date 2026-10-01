package persistence

import (
	"context"
	"errors"
)

const MaxExportRows = 10000

var ErrExportLimit = errors.New("observation export exceeds 10000 rows")

// ExportObservationAudit includes invalidated facts. Fetch one extra row so a
// bounded export can fail explicitly instead of silently losing audit history.
func (s *Store) ExportObservationAudit(ctx context.Context, listing string) ([]ObservationAudit, error) {
	rows, err := s.db.QueryContext(ctx, auditSelect+`WHERE o.listing_id=$1 ORDER BY o.observed_at ASC,o.observed_at_ns_remainder ASC,o.result_id COLLATE "C" ASC LIMIT $2`, listing, MaxExportRows+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]ObservationAudit, 0)
	for rows.Next() {
		a, err := scanAudit(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, a)
		if len(values) > MaxExportRows {
			return nil, ErrExportLimit
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}
