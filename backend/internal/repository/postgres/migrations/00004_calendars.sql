-- +goose Up
-- Calendários de mundos ficcionais: eras em sequência, cada uma contando os anos do 1.
-- eras é uma lista de {id, name, abbr, years}; years nulo só na última (era aberta).
CREATE TABLE calendars (
    id          text PRIMARY KEY,
    owner_id    text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    shared      boolean NOT NULL DEFAULT false,
    eras        jsonb NOT NULL,
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL
);
CREATE INDEX calendars_owner_idx ON calendars (owner_id);

-- Sem calendário, a timeline usa o gregoriano. Um calendário em uso não pode ser apagado.
ALTER TABLE timelines ADD COLUMN calendar_id text REFERENCES calendars (id) ON DELETE RESTRICT;
CREATE INDEX timelines_calendar_idx ON timelines (calendar_id) WHERE calendar_id IS NOT NULL;

-- +goose Down
ALTER TABLE timelines DROP COLUMN calendar_id;
DROP TABLE calendars;
