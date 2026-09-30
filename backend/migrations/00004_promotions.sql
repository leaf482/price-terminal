-- +goose Up
CREATE TABLE promotions (
    id text PRIMARY KEY CHECK (id ~ '[^[:space:]]'),
    listing_id text NOT NULL REFERENCES listings(id),
    source text NOT NULL CHECK (source ~ '[^[:space:]]'),
    observed_at timestamptz NOT NULL CHECK (isfinite(observed_at)),
    observed_ns smallint NOT NULL CHECK (observed_ns BETWEEN 0 AND 999),
    starts_at timestamptz CHECK (isfinite(starts_at)),
    starts_ns smallint CHECK (starts_ns BETWEEN 0 AND 999),
    ends_at timestamptz CHECK (isfinite(ends_at)),
    ends_ns smallint CHECK (ends_ns BETWEEN 0 AND 999),
    kind text NOT NULL CHECK (kind IN ('fixed','percentage','cashback','membership')),
    amount bigint CHECK (amount >= 0),
    currency text CHECK (currency IN ('USD','JPY')),
    basis_points bigint CHECK (basis_points BETWEEN 1 AND 10000),
    requirement text NOT NULL CHECK (requirement IN ('none','membership','other','unknown')),
    stacking text NOT NULL CHECK (stacking IN ('allowed','disallowed','unknown')),
    terms text NOT NULL CHECK (terms ~ '[^[:space:]]'),
    CHECK ((starts_at IS NULL) = (starts_ns IS NULL)),
    CHECK ((ends_at IS NULL) = (ends_ns IS NULL)),
    CHECK (starts_at IS NULL OR ends_at IS NULL OR (ends_at,ends_ns) > (starts_at,starts_ns)),
    CHECK ((kind = 'percentage' AND basis_points IS NOT NULL AND amount IS NULL AND currency IS NULL)
        OR (kind <> 'percentage' AND basis_points IS NULL AND amount IS NOT NULL AND currency IS NOT NULL)),
    CHECK (kind <> 'membership' OR requirement = 'membership')
);
CREATE INDEX promotions_listing_time_idx ON promotions(listing_id, observed_at, observed_ns, id COLLATE "C");

-- +goose Down
DROP TABLE promotions;
