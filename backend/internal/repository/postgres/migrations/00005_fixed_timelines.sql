-- +goose Up
-- Timelines fixas (ex.: ciclo de vida do lagarto): sem calendário, os eventos ficam a
-- tantos dias, semanas, meses ou anos do início. events.year guarda essa posição.
ALTER TABLE timelines
    ADD COLUMN scale text CHECK (scale IN ('dias', 'semanas', 'meses', 'anos')),
    ADD COLUMN cyclic boolean NOT NULL DEFAULT false,
    ADD CONSTRAINT timelines_scale_or_calendar CHECK (scale IS NULL OR calendar_id IS NULL);

-- +goose Down
ALTER TABLE timelines
    DROP CONSTRAINT timelines_scale_or_calendar,
    DROP COLUMN cyclic,
    DROP COLUMN scale;
