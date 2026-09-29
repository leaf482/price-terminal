-- +goose Up
CREATE TABLE products (
    id text PRIMARY KEY CHECK (id ~ '[^[:space:]]'),
    name text NOT NULL DEFAULT '',
    brand text NOT NULL DEFAULT '',
    model text NOT NULL DEFAULT ''
);

CREATE TABLE retailers (
    id text PRIMARY KEY CHECK (id ~ '[^[:space:]]'),
    name text NOT NULL DEFAULT ''
);

CREATE TABLE listings (
    id text PRIMARY KEY CHECK (id ~ '[^[:space:]]'),
    product_id text NOT NULL REFERENCES products (id),
    retailer_id text NOT NULL REFERENCES retailers (id),
    url text NOT NULL CHECK (url ~ '[^[:space:]]'),
    retailer_product_id text NOT NULL DEFAULT '',
    -- Exact source identity only: an empty retailer_product_id means absent.
    -- Different variant references on one URL remain distinct. Product ID is
    -- excluded so a duplicate source cannot be attached to a different product.
    -- URL aliases and retailer identifiers across URLs are not inferred matches.
    CONSTRAINT listings_source_identity_key UNIQUE (retailer_id, url, retailer_product_id)
);

-- +goose Down
DROP TABLE listings;
DROP TABLE retailers;
DROP TABLE products;
