-- +goose Up
-- citext stores text with case-insensitive comparisons while preserving the
-- original casing.
CREATE EXTENSION IF NOT EXISTS citext;

-- +goose Down
DROP EXTENSION IF EXISTS citext;
