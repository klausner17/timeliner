-- +goose Up
-- Links a mais de um evento, além do source_url: [{"url": "...", "label": "..."}].
-- Só os endereços ficam aqui; a prévia (título, resumo, miniatura) vem da página na hora.
ALTER TABLE events ADD COLUMN links jsonb NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE events DROP COLUMN links;
