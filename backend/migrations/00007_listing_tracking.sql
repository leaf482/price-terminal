-- +goose Up
ALTER TABLE listings ADD COLUMN tracking_enabled boolean NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE listings DROP COLUMN tracking_enabled;
