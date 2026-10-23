package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

type follows struct{ *db }

func (r follows) exec(ctx context.Context, sql string, args ...any) error {
	_, err := r.q(ctx).Exec(ctx, sql, args...)
	return translate(err)
}

func (r follows) ids(ctx context.Context, sql, id string) ([]string, error) {
	rows, err := r.q(ctx).Query(ctx, sql, id)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
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

func (r follows) FollowedTimelines(ctx context.Context, u string) ([]string, error) {
	return r.ids(ctx, `SELECT timeline_id FROM timeline_follows WHERE user_id=$1 ORDER BY id DESC`, u)
}

func (r follows) FollowedUsers(ctx context.Context, u string) ([]string, error) {
	return r.ids(ctx, `SELECT followed_id FROM user_follows WHERE follower_id=$1 ORDER BY id DESC`, u)
}

type comparisons struct{ *db }

func (r comparisons) Create(ctx context.Context, c *domain.SavedComparison) error {
	_, err := r.q(ctx).Exec(ctx, `INSERT INTO saved_comparisons (id, user_id, slugs, start_year, end_year, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`, c.ID, c.UserID, c.Slugs, c.Start, c.End, c.CreatedAt)
	return translate(err)
}

func (r comparisons) ListByUser(ctx context.Context, userID string) ([]domain.SavedComparison, error) {
	rows, err := r.q(ctx).Query(ctx, `SELECT id, user_id, slugs, start_year, end_year, created_at
		FROM saved_comparisons WHERE user_id=$1 ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.SavedComparison, error) {
		var c domain.SavedComparison
		return c, row.Scan(&c.ID, &c.UserID, &c.Slugs, &c.Start, &c.End, &c.CreatedAt)
	})
}

func (r comparisons) Delete(ctx context.Context, id, userID string) error {
	return mustAffect(r.q(ctx).Exec(ctx, `DELETE FROM saved_comparisons WHERE id=$1 AND user_id=$2`, id, userID))
}
