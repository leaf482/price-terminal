-- Proof of migration execution only; no application schema is introduced.
-- +goose Up
SELECT 1;

-- +goose Down
SELECT 1;
