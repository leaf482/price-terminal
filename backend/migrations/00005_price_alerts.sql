-- +goose Up
CREATE TABLE price_alerts (
    id text PRIMARY KEY CHECK (id ~ '[^[:space:]]'),
    listing_id text NOT NULL REFERENCES listings(id),
    kind text NOT NULL CHECK (kind IN ('target','drop','historical_low')),
    currency text NOT NULL CHECK (currency IN ('USD','JPY')),
    threshold_minor_units bigint CHECK (threshold_minor_units >= 0),
    drop_basis_points bigint CHECK (drop_basis_points BETWEEN 1 AND 10000),
    enabled boolean NOT NULL,
    require_in_stock boolean NOT NULL,
    CHECK ((kind='target' AND threshold_minor_units IS NOT NULL AND drop_basis_points IS NULL)
        OR (kind='drop' AND threshold_minor_units IS NULL AND drop_basis_points IS NOT NULL)
        OR (kind='historical_low' AND threshold_minor_units IS NULL AND drop_basis_points IS NULL))
);
CREATE INDEX price_alerts_listing_idx ON price_alerts(listing_id,id COLLATE "C");
CREATE TABLE price_alert_events (
    alert_id text NOT NULL REFERENCES price_alerts(id),
    listing_id text NOT NULL REFERENCES listings(id),
    observation_id text NOT NULL REFERENCES price_observations(result_id),
    kind text NOT NULL CHECK (kind IN ('target','drop','historical_low')),
    triggered_at timestamptz NOT NULL CHECK (isfinite(triggered_at)),
    minor_units bigint NOT NULL CHECK (minor_units >= 0),
    currency text NOT NULL CHECK (currency IN ('USD','JPY')),
    price_basis text NOT NULL CHECK (price_basis IN ('offer_price','sale_price')),
    comparison_minor_units bigint CHECK (comparison_minor_units >= 0),
    PRIMARY KEY (alert_id,observation_id)
);
CREATE INDEX price_alert_events_listing_idx ON price_alert_events(listing_id,triggered_at DESC,alert_id COLLATE "C",observation_id COLLATE "C");
-- +goose Down
DROP TABLE price_alert_events;
DROP TABLE price_alerts;
