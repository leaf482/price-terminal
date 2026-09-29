-- +goose Up
CREATE TABLE price_observations (
    -- Caller assigns one globally unique ID per collected result, reused on retries.
    result_id text PRIMARY KEY CHECK (result_id ~ '[^[:space:]]'),
    listing_id text NOT NULL REFERENCES listings (id),
    observed_at timestamptz NOT NULL CHECK (isfinite(observed_at)),
    -- PostgreSQL stores microseconds; retain the remaining Go nanoseconds exactly.
    observed_at_ns_remainder smallint NOT NULL CHECK (observed_at_ns_remainder BETWEEN 0 AND 999),
    source text NOT NULL CHECK (source ~ '[^[:space:]]'),
    stock text NOT NULL CHECK (stock IN ('unknown', 'in_stock', 'out_of_stock')),
    currency text CHECK (currency IN ('USD', 'JPY')),
    msrp bigint CHECK (msrp >= 0),
    retailer_list_price bigint CHECK (retailer_list_price >= 0),
    sale_price bigint CHECK (sale_price >= 0),
    offer_price bigint CHECK (offer_price >= 0),
    msrp_source text NOT NULL,
    CHECK ((currency IS NOT NULL) =
        (msrp IS NOT NULL OR retailer_list_price IS NOT NULL OR sale_price IS NOT NULL OR offer_price IS NOT NULL)),
    CHECK ((msrp IS NOT NULL) = (msrp_source ~ '[^[:space:]]'))
);

CREATE INDEX price_observations_history_idx ON price_observations
    (listing_id, observed_at, observed_at_ns_remainder, result_id COLLATE "C");

-- +goose Down
DROP TABLE price_observations;
