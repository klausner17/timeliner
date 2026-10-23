-- +goose Up
-- Cobre a ordem cronológica inteira, para a paginação por cursor não precisar ordenar.
CREATE INDEX events_timeline_order_idx ON events (timeline_id, year, (COALESCE(end_year, year)), created_at, id);
DROP INDEX events_timeline_year_idx;

-- +goose Down
CREATE INDEX events_timeline_year_idx ON events (timeline_id, year);
DROP INDEX events_timeline_order_idx;
