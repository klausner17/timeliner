-- +goose Up
-- Favoritar é a estrela das timelines: um voto leve, separado de seguir. O total fica na
-- timeline (como followers_count) e ordena o "Populares" do explorar.
CREATE TABLE timeline_favorites (
    user_id     text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    timeline_id text NOT NULL REFERENCES timelines (id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    id          bigserial,
    PRIMARY KEY (user_id, timeline_id)
);
CREATE INDEX timeline_favorites_timeline_idx ON timeline_favorites (timeline_id);

ALTER TABLE timelines ADD COLUMN favorites_count integer NOT NULL DEFAULT 0;

-- +goose StatementBegin
CREATE FUNCTION timeline_favorites_count() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE timelines SET favorites_count = favorites_count + 1 WHERE id = NEW.timeline_id;
    ELSE
        UPDATE timelines SET favorites_count = favorites_count - 1 WHERE id = OLD.timeline_id;
    END IF;
    RETURN NULL;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER timeline_favorites_count AFTER INSERT OR DELETE ON timeline_favorites
    FOR EACH ROW EXECUTE FUNCTION timeline_favorites_count();

-- "Populares" passa a ser favoritos, depois seguidores, depois as mais novas.
DROP INDEX timelines_listed_popular_idx;
CREATE INDEX timelines_listed_popular_idx ON timelines (favorites_count DESC, followers_count DESC, created_at DESC, id)
    WHERE visibility = 'publica' AND NOT draft;

-- +goose Down
DROP INDEX timelines_listed_popular_idx;
CREATE INDEX timelines_listed_popular_idx ON timelines (followers_count DESC, created_at DESC, id)
    WHERE visibility = 'publica' AND NOT draft;
DROP TRIGGER timeline_favorites_count ON timeline_favorites;
DROP FUNCTION timeline_favorites_count();
ALTER TABLE timelines DROP COLUMN favorites_count;
DROP TABLE timeline_favorites;
