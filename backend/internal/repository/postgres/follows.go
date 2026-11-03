package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

type follows struct{ *db }

func (r follows) exec(ctx context.Context, sql string, args ...any) error {
	_, err := r.q(ctx).Exec(ctx, sql, args...)
	return translate(err)
}

func (r follows) FollowTimeline(ctx context.Context, u, t string) error {
	return r.exec(ctx, `INSERT INTO timeline_follows (user_id, timeline_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, u, t)
}

func (r follows) UnfollowTimeline(ctx context.Context, u, t string) error {
	return r.exec(ctx, `DELETE FROM timeline_follows WHERE user_id=$1 AND timeline_id=$2`, u, t)
}

func (r follows) FollowUser(ctx context.Context, a, b string) error {
	return r.exec(ctx, `INSERT INTO user_follows (follower_id, followed_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, a, b)
}

func (r follows) UnfollowUser(ctx context.Context, a, b string) error {
	return r.exec(ctx, `DELETE FROM user_follows WHERE follower_id=$1 AND followed_id=$2`, a, b)
}

func (r follows) FollowedUsers(ctx context.Context, u string, p usecase.Paging) ([]string, int, error) {
	var total int
	if err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM user_follows WHERE follower_id=$1`, u).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.q(ctx).Query(ctx, `SELECT followed_id FROM user_follows WHERE follower_id=$1 ORDER BY id DESC LIMIT $2 OFFSET $3`, u, p.Limit, p.Offset)
	if err != nil {
		return nil, 0, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	return ids, total, err
}

type comparisons struct{ *db }

func (r comparisons) Create(ctx context.Context, c *domain.SavedComparison) error {
	_, err := r.q(ctx).Exec(ctx, `INSERT INTO saved_comparisons (id, user_id, slugs, start_year, end_year, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`, c.ID, c.UserID, c.Slugs, c.Start, c.End, c.CreatedAt)
	return translate(err)
}

func (r comparisons) ListByUser(ctx context.Context, userID string, slugs []string, p usecase.Paging) ([]domain.SavedComparison, int, error) {
	cond := `user_id=$1 AND (cardinality($2::text[]) = 0 OR slugs = $2::text[])`
	if slugs == nil {
		slugs = []string{}
	}
	var total int
	if err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM saved_comparisons WHERE `+cond, userID, slugs).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.q(ctx).Query(ctx, `SELECT id, user_id, slugs, start_year, end_year, created_at
		FROM saved_comparisons WHERE `+cond+` ORDER BY created_at DESC, id DESC LIMIT $3 OFFSET $4`, userID, slugs, p.Limit, p.Offset)
	if err != nil {
		return nil, 0, err
	}
	cs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.SavedComparison, error) {
		var c domain.SavedComparison
		return c, row.Scan(&c.ID, &c.UserID, &c.Slugs, &c.Start, &c.End, &c.CreatedAt)
	})
	return cs, total, err
}

func (r comparisons) Delete(ctx context.Context, id, userID string) error {
	return mustAffect(r.q(ctx).Exec(ctx, `DELETE FROM saved_comparisons WHERE id=$1 AND user_id=$2`, id, userID))
}
