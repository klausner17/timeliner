-- +goose Up
-- Busca do explorar: ILIKE '%texto%' em título, descrição e tags sem varrer a tabela.
-- pg_trgm é uma extensão "trusted" desde o Postgres 13: o dono do banco pode criá-la.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- array_to_string não é IMMUTABLE, então o texto da busca passa por esta função para
-- poder virar índice. A consulta usa a mesma expressão.
-- +goose StatementBegin
CREATE FUNCTION timeline_search_text(title text, description text, tags text[]) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE
AS $$ SELECT title || ' ' || description || ' ' || array_to_string(tags, ' ') $$;
-- +goose StatementEnd
CREATE INDEX timelines_search_idx ON timelines USING gin (timeline_search_text(title, description, tags) gin_trgm_ops);

-- "Populares" ordenava contando os seguidores de cada timeline a cada página. O total
-- fica guardado na timeline e um gatilho o mantém em dia.
ALTER TABLE timelines ADD COLUMN followers_count integer NOT NULL DEFAULT 0;
UPDATE timelines t SET followers_count = (SELECT count(*) FROM timeline_follows f WHERE f.timeline_id = t.id);

-- +goose StatementBegin
CREATE FUNCTION timeline_followers_count() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE timelines SET followers_count = followers_count + 1 WHERE id = NEW.timeline_id;
    ELSE
        UPDATE timelines SET followers_count = followers_count - 1 WHERE id = OLD.timeline_id;
    END IF;
    RETURN NULL;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER timeline_follows_count AFTER INSERT OR DELETE ON timeline_follows
    FOR EACH ROW EXECUTE FUNCTION timeline_followers_count();

CREATE INDEX timelines_listed_popular_idx ON timelines (followers_count DESC, created_at DESC, id)
    WHERE visibility = 'publica' AND NOT draft;

-- +goose Down
DROP INDEX timelines_listed_popular_idx;
DROP TRIGGER timeline_follows_count ON timeline_follows;
DROP FUNCTION timeline_followers_count();
ALTER TABLE timelines DROP COLUMN followers_count;
DROP INDEX timelines_search_idx;
DROP FUNCTION timeline_search_text(text, text, text[]);
