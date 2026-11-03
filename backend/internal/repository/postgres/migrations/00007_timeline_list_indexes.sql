-- +goose Up
-- O explorar e o seletor do comparar paginam as timelines públicas pelas mais recentes;
-- o filtro de escala acha as fixas sem varrer a tabela.
CREATE INDEX timelines_listed_recent_idx ON timelines (created_at DESC, id) WHERE visibility = 'publica' AND NOT draft;
CREATE INDEX timelines_scale_idx ON timelines (scale) WHERE scale IS NOT NULL;

-- +goose Down
DROP INDEX timelines_scale_idx;
DROP INDEX timelines_listed_recent_idx;
