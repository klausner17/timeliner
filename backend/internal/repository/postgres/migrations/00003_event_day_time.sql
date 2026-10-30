-- +goose Up
ALTER TABLE events ADD COLUMN month smallint, ADD COLUMN day smallint, ADD COLUMN time_of_day text;

-- Datas por extenso como "13 de maio de 1888", "1º de julho de 1994" ou "Fevereiro de 1922",
-- no mesmo ano do evento, viram mês e dia. As outras se perdem com a coluna date.
WITH parsed AS (
    SELECT events.id,
           -- ano astronômico: 1 a.C. (-1) é o ano 0, para a regra do bissexto
           CASE WHEN events.year < 0 THEN events.year + 1 ELSE events.year END AS astro,
           m[1]::smallint AS day,
           array_position(ARRAY['janeiro', 'fevereiro', 'março', 'abril', 'maio', 'junho', 'julho',
                                'agosto', 'setembro', 'outubro', 'novembro', 'dezembro'], lower(m[2]))::smallint AS month
    FROM events,
         regexp_match(trim(date), '^(?:(\d{1,2})º? de )?(janeiro|fevereiro|março|abril|maio|junho|julho|agosto|setembro|outubro|novembro|dezembro) de (-?\d{1,6})$', 'i') AS m
    WHERE m[3]::int = events.year
)
UPDATE events SET month = parsed.month, day = parsed.day
FROM parsed
WHERE events.id = parsed.id
  AND (parsed.day IS NULL OR parsed.day BETWEEN 1 AND CASE
        WHEN parsed.month = 2 AND parsed.astro % 4 = 0 AND (parsed.astro % 100 <> 0 OR parsed.astro % 400 = 0) THEN 29
        WHEN parsed.month = 2 THEN 28
        WHEN parsed.month IN (4, 6, 9, 11) THEN 30
        ELSE 31
      END);

CREATE INDEX events_timeline_order_v2_idx ON events (timeline_id, year, (COALESCE(month, 0)), (COALESCE(day, 0)),
    (COALESCE(time_of_day, '')), (COALESCE(end_year, year)), created_at, id);
DROP INDEX events_timeline_order_idx;
ALTER INDEX events_timeline_order_v2_idx RENAME TO events_timeline_order_idx;

ALTER TABLE events DROP COLUMN date;
UPDATE suggestions SET proposed = proposed - 'date', previous = previous - 'date'
WHERE proposed ? 'date' OR previous ? 'date';

-- +goose Down
ALTER TABLE events ADD COLUMN date text;
CREATE INDEX events_timeline_order_v1_idx ON events (timeline_id, year, (COALESCE(end_year, year)), created_at, id);
DROP INDEX events_timeline_order_idx;
ALTER INDEX events_timeline_order_v1_idx RENAME TO events_timeline_order_idx;
ALTER TABLE events DROP COLUMN month, DROP COLUMN day, DROP COLUMN time_of_day;
