-- +goose Up
CREATE TABLE observation_invalidations (
    observation_id text PRIMARY KEY REFERENCES price_observations(result_id),
    reason text NOT NULL CHECK (reason ~ '[^[:space:]]' AND length(reason) <= 1000),
    invalidated_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK (isfinite(invalidated_at))
);
-- Product current-price and catalog reads filter by Product and order by Listing.
CREATE INDEX listings_product_id_idx ON listings(product_id,id COLLATE "C");
-- +goose Down
DROP INDEX listings_product_id_idx;
DROP TABLE observation_invalidations;
