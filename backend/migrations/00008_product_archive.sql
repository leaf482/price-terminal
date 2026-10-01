-- +goose Up
ALTER TABLE products ADD COLUMN archived boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE products DROP COLUMN archived;
