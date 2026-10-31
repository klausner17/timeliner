package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

type calendars struct{ *db }

type calendarEraJSON struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Abbr  string `json:"abbr"`
	Years *int   `json:"years,omitempty"`
}

const calendarCols = `id, owner_id, name, description, shared, eras, created_at, updated_at`

func scanCalendar(row pgx.Row) (*domain.Calendar, error) {
	var c domain.Calendar
	var eras []byte
	if err := row.Scan(&c.ID, &c.OwnerID, &c.Name, &c.Description, &c.Shared, &eras, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, translate(err)
	}
	var ej []calendarEraJSON
	if err := json.Unmarshal(eras, &ej); err != nil {
		return nil, err
	}
	c.Eras = make([]domain.CalendarEra, len(ej))
	for i, e := range ej {
		c.Eras[i] = domain.CalendarEra(e)
	}
	return &c, nil
}

func calendarErasJSON(c *domain.Calendar) []byte {
	out := make([]calendarEraJSON, len(c.Eras))
	for i, e := range c.Eras {
		out[i] = calendarEraJSON(e)
	}
	b, _ := json.Marshal(out)
	return b
}

func (r calendars) Create(ctx context.Context, c *domain.Calendar) error {
	_, err := r.q(ctx).Exec(ctx, `INSERT INTO calendars (`+calendarCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		c.ID, c.OwnerID, c.Name, c.Description, c.Shared, calendarErasJSON(c), c.CreatedAt, c.UpdatedAt)
	return translate(err)
}

func (r calendars) Update(ctx context.Context, c *domain.Calendar) error {
	return mustAffect(r.q(ctx).Exec(ctx, `UPDATE calendars SET name=$2, description=$3, shared=$4, eras=$5, updated_at=$6
		WHERE id=$1`, c.ID, c.Name, c.Description, c.Shared, calendarErasJSON(c), c.UpdatedAt))
}

func (r calendars) Delete(ctx context.Context, id string) error {
	return mustAffect(r.q(ctx).Exec(ctx, `DELETE FROM calendars WHERE id=$1`, id))
}

func (r calendars) ByID(ctx context.Context, id string) (*domain.Calendar, error) {
	return scanCalendar(r.q(ctx).QueryRow(ctx, `SELECT `+calendarCols+` FROM calendars WHERE id=$1`, id))
}

func (r calendars) list(ctx context.Context, sql string, args ...any) ([]domain.Calendar, error) {
	rows, err := r.q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Calendar{}
	for rows.Next() {
		c, err := scanCalendar(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r calendars) ByIDs(ctx context.Context, ids []string) (map[string]domain.Calendar, error) {
	cs, err := r.list(ctx, `SELECT `+calendarCols+` FROM calendars WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.Calendar, len(cs))
	for _, c := range cs {
		out[c.ID] = c
	}
	return out, nil
}

func (r calendars) Usable(ctx context.Context, userID, query string) ([]domain.Calendar, error) {
	return r.list(ctx, `SELECT `+calendarCols+` FROM calendars
		WHERE (shared OR ($1 <> '' AND owner_id = $1)) AND name ILIKE $2
		ORDER BY lower(name), id`, userID, "%"+escapeLike(query)+"%")
}

func (r calendars) InUse(ctx context.Context, id string) (bool, error) {
	var used bool
	err := r.q(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM timelines WHERE calendar_id=$1)`, id).Scan(&used)
	return used, err
}

// shiftArgs separa os deslocamentos em três listas para o unnest.
func shiftArgs(shifts []domain.YearShift) ([]int, []int, []int) {
	from, to, delta := make([]int, len(shifts)), make([]int, len(shifts)), make([]int, len(shifts))
	for i, s := range shifts {
		from[i], to[i], delta[i] = s.From, s.To, s.Delta
	}
	return from, to, delta
}
