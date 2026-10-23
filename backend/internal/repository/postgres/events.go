package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

type events struct{ *db }

const eventCols = `id, timeline_id, year, end_year, date, title, description, source_url, era, author_id, created_at`

func scanEvent(row pgx.Row) (*domain.Event, error) {
	var e domain.Event
	err := row.Scan(&e.ID, &e.TimelineID, &e.Year, &e.EndYear, &e.Date, &e.Title, &e.Description,
		&e.SourceURL, &e.Era, &e.AuthorID, &e.CreatedAt)
	if err != nil {
		return nil, translate(err)
	}
	return &e, nil
}

func (r events) Create(ctx context.Context, e *domain.Event) error {
	_, err := r.q(ctx).Exec(ctx, `INSERT INTO events (`+eventCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		e.ID, e.TimelineID, e.Year, e.EndYear, e.Date, e.Title, e.Description, e.SourceURL, e.Era, e.AuthorID, e.CreatedAt)
	return translate(err)
}

func (r events) Update(ctx context.Context, e *domain.Event) error {
	return mustAffect(r.q(ctx).Exec(ctx, `UPDATE events SET year=$2, end_year=$3, date=$4, title=$5,
		description=$6, source_url=$7, era=$8 WHERE id=$1`,
		e.ID, e.Year, e.EndYear, e.Date, e.Title, e.Description, e.SourceURL, e.Era))
}

func (r events) Delete(ctx context.Context, id string) error {
	return mustAffect(r.q(ctx).Exec(ctx, `DELETE FROM events WHERE id=$1`, id))
}

func (r events) ByID(ctx context.Context, id string) (*domain.Event, error) {
	return scanEvent(r.q(ctx).QueryRow(ctx, `SELECT `+eventCols+` FROM events WHERE id=$1`, id))
}

func (r events) ListByTimeline(ctx context.Context, timelineID string, from, to *int) ([]domain.Event, error) {
	rows, err := r.q(ctx).Query(ctx, `SELECT `+eventCols+` FROM events
		WHERE timeline_id=$1 AND ($2::int IS NULL OR COALESCE(end_year, year) >= $2) AND ($3::int IS NULL OR year <= $3)
		ORDER BY year, COALESCE(end_year, year), created_at, id`, timelineID, from, to)
	if err != nil {
		return nil, err
	}
	return collect(rows, scanEvent)
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
	EndYear     *int       `json:"endYear,omitempty"`
	Date        *string    `json:"date,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	SourceURL   *string    `json:"sourceUrl,omitempty"`
	Era         *string    `json:"era,omitempty"`
}

func dataToJSON(d domain.EventData) eventJSON {
	return eventJSON{Year: d.Year, EndYear: d.EndYear, Date: d.Date, Title: d.Title, Description: d.Description, SourceURL: d.SourceURL, Era: d.Era}
}

func (j eventJSON) data() domain.EventData {
	return domain.EventData{Year: j.Year, EndYear: j.EndYear, Date: j.Date, Title: j.Title, Description: j.Description, SourceURL: j.SourceURL, Era: j.Era}
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
