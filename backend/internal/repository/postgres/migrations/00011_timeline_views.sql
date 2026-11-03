-- +goose Up
-- Visualizações (só aparecem na tela, não mudam o Populares): uma por pessoa por dia em cada timeline. Quem está logado conta pelo
-- usuário; visitante, por um hash do IP com o dia (o IP em si não fica guardado).
CREATE TABLE timeline_views (
    timeline_id text NOT NULL REFERENCES timelines (id) ON DELETE CASCADE,
    viewer      text NOT NULL,
    day         date NOT NULL,
    PRIMARY KEY (timeline_id, day, viewer)
);

-- O total fica na timeline. O gatilho só soma: apagar linhas antigas de timeline_views
-- para economizar espaço não mexe no total.
ALTER TABLE timelines ADD COLUMN views_count integer NOT NULL DEFAULT 0;

-- +goose StatementBegin
CREATE FUNCTION timeline_views_count() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE timelines SET views_count = views_count + 1 WHERE id = NEW.timeline_id;
    RETURN NULL;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER timeline_views_count AFTER INSERT ON timeline_views
    FOR EACH ROW EXECUTE FUNCTION timeline_views_count();

-- +goose Down
DROP TRIGGER timeline_views_count ON timeline_views;
DROP FUNCTION timeline_views_count();
ALTER TABLE timelines DROP COLUMN views_count;
DROP TABLE timeline_views;
