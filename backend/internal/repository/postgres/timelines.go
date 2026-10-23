package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

type timelines struct{ *db }

type eraJSON struct {
	Name  string `json:"name"`
	Start int    `json:"start"`
	End   *int   `json:"end,omitempty"`
}

func erasToJSON(eras []domain.Era) []byte {
	out := make([]eraJSON, len(eras))
	for i, e := range eras {
		out[i] = eraJSON(e)
	}
	b, _ := json.Marshal(out)
	return b
}

func tagsOf(t *domain.Timeline) []string {
	if t.Tags == nil {
		return []string{}
	}
	return t.Tags
}

const timelineCols = `t.id, t.slug, t.title, t.description, t.category, t.color, t.owner_id, t.visibility,
	t.contributions, t.draft, t.eras, t.tags, t.created_at, t.updated_at`

func scanTimeline(row pgx.Row) (*domain.Timeline, error) {
	var t domain.Timeline
	var eras []byte
	err := row.Scan(&t.ID, &t.Slug, &t.Title, &t.Description, &t.Category, &t.Color, &t.OwnerID, &t.Visibility,
		&t.Contributions, &t.Draft, &eras, &t.Tags, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, translate(err)
	}
	var ej []eraJSON
	if err := json.Unmarshal(eras, &ej); err != nil {
		return nil, err
	}
	t.Eras = make([]domain.Era, len(ej))
	for i, e := range ej {
		t.Eras[i] = domain.Era(e)
	}
	return &t, nil
}

func (r timelines) Create(ctx context.Context, t *domain.Timeline) error {
	_, err := r.q(ctx).Exec(ctx, `INSERT INTO timelines (id, slug, title, description, category, color, owner_id,
		visibility, contributions, draft, eras, tags, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		t.ID, t.Slug, t.Title, t.Description, t.Category, t.Color, t.OwnerID, t.Visibility, t.Contributions,
		t.Draft, erasToJSON(t.Eras), tagsOf(t), t.CreatedAt, t.UpdatedAt)
	return translate(err)
}

func (r timelines) Update(ctx context.Context, t *domain.Timeline) error {
	return mustAffect(r.q(ctx).Exec(ctx, `UPDATE timelines SET title=$2, description=$3, category=$4, color=$5,
		visibility=$6, contributions=$7, draft=$8, eras=$9, tags=$10, updated_at=$11 WHERE id=$1`,
		t.ID, t.Title, t.Description, t.Category, t.Color, t.Visibility, t.Contributions, t.Draft,
		erasToJSON(t.Eras), tagsOf(t), t.UpdatedAt))
}

func (r timelines) Delete(ctx context.Context, id string) error {
	return mustAffect(r.q(ctx).Exec(ctx, `DELETE FROM timelines WHERE id=$1`, id))
}

func (r timelines) ByID(ctx context.Context, id string) (*domain.Timeline, error) {
	return scanTimeline(r.q(ctx).QueryRow(ctx, `SELECT `+timelineCols+` FROM timelines t WHERE t.id=$1`, id))
}

func (r timelines) BySlug(ctx context.Context, slug string) (*domain.Timeline, error) {
	return scanTimeline(r.q(ctx).QueryRow(ctx, `SELECT `+timelineCols+` FROM timelines t WHERE t.slug=$1`, slug))
}

func (r timelines) SlugTaken(ctx context.Context, slug string) (bool, error) {
	var taken bool
	err := r.q(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM timelines WHERE slug=$1)`, slug).Scan(&taken)
	return taken, err
}

func (r timelines) List(ctx context.Context, f usecase.TimelineFilter) ([]domain.Timeline, int, error) {
	where := []string{"true"}
	args := []any{}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if !f.IncludeUnlisted {
		where = append(where, "t.visibility='publica' AND NOT t.draft")
	}
	if f.OwnerID != "" {
		where = append(where, "t.owner_id="+arg(f.OwnerID))
	}
	if f.Category != "" {
		where = append(where, "t.category="+arg(f.Category))
	}
	if f.Tag != "" {
		where = append(where, arg(f.Tag)+" = ANY(t.tags)")
	}
	if f.OnlyOpen {
		where = append(where, "t.contributions='sugestoes'")
	}
	if f.Query != "" {
		p := arg("%" + escapeLike(f.Query) + "%")
		where = append(where, fmt.Sprintf("(t.title ILIKE %[1]s OR t.description ILIKE %[1]s OR array_to_string(t.tags, ' ') ILIKE %[1]s)", p))
	}
	cond := strings.Join(where, " AND ")

	var total int
	if err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM timelines t WHERE `+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := "t.created_at DESC, t.id"
	if f.Sort == usecase.SortPopular {
		order = "(SELECT count(*) FROM timeline_follows tf WHERE tf.timeline_id=t.id) DESC, " + order
	}
	rows, err := r.q(ctx).Query(ctx, `SELECT `+timelineCols+` FROM timelines t WHERE `+cond+
		` ORDER BY `+order+` LIMIT `+arg(f.Limit)+` OFFSET `+arg(f.Offset), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []domain.Timeline{}
	for rows.Next() {
		t, err := scanTimeline(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *t)
	}
	return out, total, rows.Err()
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (r timelines) Stats(ctx context.Context, ids []string, viewerID string) (map[string]domain.TimelineStats, error) {
	rows, err := r.q(ctx).Query(ctx, `SELECT t.id,
		(SELECT count(*) FROM events e WHERE e.timeline_id=t.id),
		(SELECT count(DISTINCT e.author_id) FROM events e WHERE e.timeline_id=t.id AND e.author_id<>t.owner_id),
		(SELECT count(*) FROM suggestions s WHERE s.timeline_id=t.id AND s.status='pendente'),
		(SELECT count(*) FROM timeline_follows f WHERE f.timeline_id=t.id),
		EXISTS (SELECT 1 FROM timeline_follows f WHERE f.timeline_id=t.id AND f.user_id=$2),
		COALESCE((SELECT array_agg(e.year ORDER BY e.year) FROM events e WHERE e.timeline_id=t.id), '{}'),
		(SELECT min(e.year) FROM events e WHERE e.timeline_id=t.id),
		(SELECT max(COALESCE(e.end_year, e.year)) FROM events e WHERE e.timeline_id=t.id)
		FROM timelines t WHERE t.id = ANY($1)`, ids, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]domain.TimelineStats{}
	for rows.Next() {
		var id string
		var s domain.TimelineStats
		if err := rows.Scan(&id, &s.EventCount, &s.ContributorCount, &s.PendingSuggestions, &s.Followers,
			&s.Following, &s.Years, &s.StartYear, &s.EndYear); err != nil {
			return nil, err
		}
		out[id] = s
	}
	return out, rows.Err()
}

func (r timelines) Tags(ctx context.Context, limit int) ([]domain.TagCount, error) {
	rows, err := r.q(ctx).Query(ctx, `SELECT tag, count(*) FROM timelines t, unnest(t.tags) AS tag
		WHERE t.visibility='publica' AND NOT t.draft GROUP BY tag ORDER BY count(*) DESC, tag LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.TagCount, error) {
		var t domain.TagCount
		return t, row.Scan(&t.Tag, &t.Count)
	})
}
