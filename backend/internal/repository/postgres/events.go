package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

type events struct{ *db }

const eventCols = `id, timeline_id, year, month, day, time_of_day, end_year, title, description, source_url, era, author_id, created_at`

// eventFields são os destinos do Scan na ordem de eventCols.
func eventFields(e *domain.Event) []any {
	return []any{&e.ID, &e.TimelineID, &e.Year, &e.Month, &e.Day, &e.Time, &e.EndYear, &e.Title, &e.Description,
		&e.SourceURL, &e.Era, &e.AuthorID, &e.CreatedAt}
}

// eventValues são os valores do evento na ordem de eventCols.
func eventValues(e *domain.Event) []any {
	return []any{e.ID, e.TimelineID, e.Year, e.Month, e.Day, e.Time, e.EndYear, e.Title, e.Description,
		e.SourceURL, e.Era, e.AuthorID, e.CreatedAt}
}

func scanEvent(row pgx.Row) (*domain.Event, error) {
	var e domain.Event
	if err := row.Scan(eventFields(&e)...); err != nil {
		return nil, translate(err)
	}
	return &e, nil
}

func (r events) Create(ctx context.Context, e *domain.Event) error {
	_, err := r.q(ctx).Exec(ctx, `INSERT INTO events (`+eventCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		eventValues(e)...)
	return translate(err)
}

func (r events) Update(ctx context.Context, e *domain.Event) error {
	return mustAffect(r.q(ctx).Exec(ctx, `UPDATE events SET year=$2, month=$3, day=$4, time_of_day=$5, end_year=$6,
		title=$7, description=$8, source_url=$9, era=$10 WHERE id=$1`,
		e.ID, e.Year, e.Month, e.Day, e.Time, e.EndYear, e.Title, e.Description, e.SourceURL, e.Era))
}

func (r events) Delete(ctx context.Context, id string) error {
	return mustAffect(r.q(ctx).Exec(ctx, `DELETE FROM events WHERE id=$1`, id))
}

func (r events) ByID(ctx context.Context, id string) (*domain.Event, error) {
	return scanEvent(r.q(ctx).QueryRow(ctx, `SELECT `+eventCols+` FROM events WHERE id=$1`, id))
}

func (r events) CreateMany(ctx context.Context, es []domain.Event) error {
	rows := make([][]any, len(es))
	for i := range es {
		rows[i] = eventValues(&es[i])
	}
	_, err := r.q(ctx).CopyFrom(ctx, pgx.Identifier{"events"}, strings.Split(strings.ReplaceAll(eventCols, " ", ""), ","), pgx.CopyFromRows(rows))
	return translate(err)
}

// eventOrder é a ordem cronológica, igual a memory.SortEvents e ao índice events_timeline_order_idx.
const eventOrder = `year, COALESCE(month, 0), COALESCE(day, 0), COALESCE(time_of_day, ''), COALESCE(end_year, year), created_at, id`

// touches filtra os eventos da timeline $1 que tocam o intervalo [$2, $3]; nulos não filtram.
const touches = `timeline_id=$1 AND ($2::int IS NULL OR COALESCE(end_year, year) >= $2) AND ($3::int IS NULL OR year <= $3)`

func (r events) ListByTimeline(ctx context.Context, q usecase.EventQuery) ([]domain.Event, error) {
	after := []any{nil, nil, nil, nil, nil, nil, nil}
	if c := q.After; c != nil {
		after = []any{c.Year, c.Month, c.Day, c.Time, c.EndYear, c.CreatedAt, c.ID}
	}
	args := append([]any{q.TimelineID, q.From, q.To}, append(after, q.Limit)...)
	rows, err := r.q(ctx).Query(ctx, `SELECT `+eventCols+` FROM events
		WHERE `+touches+` AND ($4::int IS NULL OR (`+eventOrder+`) > ($4, $5::int, $6::int, $7::text, $8::int, $9::timestamptz, $10::text))
		ORDER BY `+eventOrder+` LIMIT $11`, args...)
	if err != nil {
		return nil, err
	}
	return collect(rows, scanEvent)
}

func (r events) Summary(ctx context.Context, q usecase.SummaryQuery) ([]domain.YearBucket, error) {
	span := q.To - q.From + 1
	// Conta por faixa e, para cada faixa com eventos, busca os primeiros pelo índice. A
	// faixa b vai de $2 + ceil(b*span/n) a $2 + ceil((b+1)*span/n) - 1, como em domain.BucketRange;
	// a primeira também leva quem começa antes de $2.
	rows, err := r.q(ctx).Query(ctx, `WITH c AS (
			SELECT b, count(*) AS n FROM (SELECT ((GREATEST(year, $2) - $2)::bigint * $4 / $5)::int AS b FROM events WHERE `+touches+`) x
			GROUP BY b)
		SELECT c.b, c.n, s.* FROM c CROSS JOIN LATERAL (
			SELECT `+eventCols+` FROM events WHERE `+touches+`
				AND year <= $2 + ((c.b + 1)::bigint * $5 + $4 - 1) / $4 - 1
				AND year >= CASE WHEN c.b = 0 THEN -2147483648 ELSE $2 + (c.b::bigint * $5 + $4 - 1) / $4 END
			ORDER BY `+eventOrder+` LIMIT $6) s
		ORDER BY c.b, s.year, COALESCE(s.month, 0), COALESCE(s.day, 0), COALESCE(s.time_of_day, ''), COALESCE(s.end_year, s.year), s.created_at, s.id`, q.TimelineID, q.From, q.To, q.Buckets, span, q.Sample)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.YearBucket{}
	last := -1
	for rows.Next() {
		var b, n int
		var e domain.Event
		if err := rows.Scan(append([]any{&b, &n}, eventFields(&e)...)...); err != nil {
			return nil, err
		}
		if b != last {
			from, to := domain.BucketRange(b, q.From, span, q.Buckets)
			out = append(out, domain.YearBucket{From: from, To: to, Count: n, Sample: []domain.Event{}})
			last = b
		}
		out[len(out)-1].Sample = append(out[len(out)-1].Sample, e)
	}
	return out, rows.Err()
}

func (r events) Overlaps(ctx context.Context, timelineIDs []string, from, to, limit int) ([]domain.YearOverlap, int, error) {
	rows, err := r.q(ctx).Query(ctx, `SELECT year, array_agg(DISTINCT timeline_id ORDER BY timeline_id), count(*) OVER ()
		FROM events WHERE timeline_id = ANY($1) AND year BETWEEN $2 AND $3
		GROUP BY year HAVING count(DISTINCT timeline_id) >= 2
		ORDER BY year LIMIT $4`, timelineIDs, from, to, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []domain.YearOverlap{}
	total := 0
	for rows.Next() {
		var o domain.YearOverlap
		if err := rows.Scan(&o.Year, &o.TimelineIDs, &total); err != nil {
			return nil, 0, err
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func collect[T any](rows pgx.Rows, scan func(pgx.Row) (*T, error)) ([]T, error) {
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

// eventJSON é o formato dos eventos guardados em jsonb nas sugestões.
type eventJSON struct {
	ID          string     `json:"id,omitempty"`
	TimelineID  string     `json:"timelineId,omitempty"`
	AuthorID    string     `json:"authorId,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	Year        int        `json:"year"`
	Month       *int       `json:"month,omitempty"`
	Day         *int       `json:"day,omitempty"`
	Time        *string    `json:"time,omitempty"`
	EndYear     *int       `json:"endYear,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	SourceURL   *string    `json:"sourceUrl,omitempty"`
	Era         *string    `json:"era,omitempty"`
}

func dataToJSON(d domain.EventData) eventJSON {
	return eventJSON{Year: d.Year, Month: d.Month, Day: d.Day, Time: d.Time, EndYear: d.EndYear, Title: d.Title, Description: d.Description, SourceURL: d.SourceURL, Era: d.Era}
}

func (j eventJSON) data() domain.EventData {
	return domain.EventData{Year: j.Year, Month: j.Month, Day: j.Day, Time: j.Time, EndYear: j.EndYear, Title: j.Title, Description: j.Description, SourceURL: j.SourceURL, Era: j.Era}
}

func marshalEvent(e *domain.Event) ([]byte, error) {
	if e == nil {
		return nil, nil
	}
	j := dataToJSON(e.EventData)
	j.ID, j.TimelineID, j.AuthorID, j.CreatedAt = e.ID, e.TimelineID, e.AuthorID, &e.CreatedAt
	return json.Marshal(j)
}

func unmarshalEvent(b []byte) (*domain.Event, error) {
	if b == nil {
		return nil, nil
	}
	var j eventJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return nil, err
	}
	e := &domain.Event{EventData: j.data(), ID: j.ID, TimelineID: j.TimelineID, AuthorID: j.AuthorID}
	if j.CreatedAt != nil {
		e.CreatedAt = *j.CreatedAt
	}
	return e, nil
}
