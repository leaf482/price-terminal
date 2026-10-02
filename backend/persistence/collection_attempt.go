package persistence

import (
	"context"
	"fmt"
	"github.com/leaf482/price-terminal/backend/domain"
)

const CollectionAttemptLimit = 50

func (s *Store) InsertCollectionAttempt(ctx context.Context, a domain.CollectionAttempt) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO collection_attempts
 (id,listing_id,trigger,started_at,finished_at,outcome,observation_id,error_summary)
 VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8)`, a.ID, a.ListingID, a.Trigger, a.StartedAt, a.FinishedAt, a.Outcome, a.ObservationID, a.ErrorSummary)
	if err != nil {
		return fmt.Errorf("insert collection attempt: %w", err)
	}
	return nil
}

func (s *Store) ListCollectionAttempts(ctx context.Context, id string) ([]domain.CollectionAttempt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,listing_id,trigger,started_at,finished_at,outcome,COALESCE(observation_id,''),error_summary
 FROM collection_attempts WHERE listing_id=$1 ORDER BY started_at DESC,id COLLATE "C" DESC LIMIT $2`, id, CollectionAttemptLimit)
	if err != nil {
		return nil, fmt.Errorf("list collection attempts: %w", err)
	}
	defer rows.Close()
	result := make([]domain.CollectionAttempt, 0)
	for rows.Next() {
		var a domain.CollectionAttempt
		if err := rows.Scan(&a.ID, &a.ListingID, &a.Trigger, &a.StartedAt, &a.FinishedAt, &a.Outcome, &a.ObservationID, &a.ErrorSummary); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
