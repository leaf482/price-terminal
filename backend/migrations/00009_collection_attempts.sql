-- +goose Up
CREATE TABLE collection_attempts (
    id text PRIMARY KEY,
    listing_id text NOT NULL REFERENCES listings(id),
    trigger text NOT NULL CHECK (trigger IN ('manual', 'scheduled')),
    started_at timestamptz NOT NULL,
    finished_at timestamptz NOT NULL CHECK (finished_at >= started_at),
    outcome text NOT NULL CHECK (outcome IN ('success', 'provider_error', 'persistence_error', 'cancelled', 'unavailable')),
    observation_id text REFERENCES price_observations(result_id),
    error_summary text NOT NULL,
    CHECK ((outcome = 'success' AND observation_id IS NOT NULL AND error_summary = '') OR
           (outcome <> 'success' AND observation_id IS NULL AND error_summary = outcome))
);
CREATE INDEX collection_attempts_listing_recent_idx ON collection_attempts(listing_id, started_at DESC, id COLLATE "C" DESC);

-- +goose Down
DROP TABLE collection_attempts;
