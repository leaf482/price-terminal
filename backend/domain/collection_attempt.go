package domain

import "time"

// CollectionAttempt contains operational metadata, never observed price facts.
// ErrorSummary is a fixed outcome code, not a provider or database error.
type CollectionAttempt struct {
	ID            string    `json:"id"`
	ListingID     string    `json:"listing_id"`
	Trigger       string    `json:"trigger"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
	Outcome       string    `json:"outcome"`
	ObservationID string    `json:"observation_id,omitempty"`
	ErrorSummary  string    `json:"error_summary,omitempty"`
}
