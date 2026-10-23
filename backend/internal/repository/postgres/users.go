package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

type users struct{ *db }

const userCols = `id, handle, name, email, password_hash, bio, tint, created_at`

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Handle, &u.Name, &u.Email, &u.PasswordHash, &u.Bio, &u.Tint, &u.CreatedAt)
	if err != nil {
		return nil, translate(err)
	}
	return &u, nil
}

func (r users) Create(ctx context.Context, u *domain.User) error {
	_, err := r.q(ctx).Exec(ctx, `INSERT INTO users (`+userCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		u.ID, u.Handle, u.Name, u.Email, u.PasswordHash, u.Bio, u.Tint, u.CreatedAt)
	return translate(err)
}

func (r users) Update(ctx context.Context, u *domain.User) error {
	return mustAffect(r.q(ctx).Exec(ctx, `UPDATE users SET name=$2, bio=$3, tint=$4, password_hash=$5 WHERE id=$1`,
		u.ID, u.Name, u.Bio, u.Tint, u.PasswordHash))
}

func (r users) one(ctx context.Context, where string, arg any) (*domain.User, error) {
	return scanUser(r.q(ctx).QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE `+where, arg))
}

func (r users) ByID(ctx context.Context, id string) (*domain.User, error) {
	return r.one(ctx, "id=$1", id)
}

func (r users) ByHandle(ctx context.Context, h string) (*domain.User, error) {
	return r.one(ctx, "handle=$1", h)
}

func (r users) ByEmail(ctx context.Context, e string) (*domain.User, error) {
	return r.one(ctx, "email=$1", e)
}

func (r users) ByIDs(ctx context.Context, ids []string) (map[string]domain.User, error) {
	rows, err := r.q(ctx).Query(ctx, `SELECT `+userCols+` FROM users WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]domain.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out[u.ID] = *u
	}
	return out, rows.Err()
}

func (r users) Stats(ctx context.Context, userID, viewerID string) (domain.UserStats, error) {
	var s domain.UserStats
	err := r.q(ctx).QueryRow(ctx, `SELECT
		(SELECT count(*) FROM timelines WHERE owner_id=$1 AND visibility='publica' AND NOT draft),
		(SELECT count(*) FROM events WHERE author_id=$1),
		(SELECT count(*) FROM suggestions WHERE author_id=$1 AND status='aprovada'),
		(SELECT count(*) FROM user_follows WHERE followed_id=$1),
		EXISTS (SELECT 1 FROM user_follows WHERE followed_id=$1 AND follower_id=$2)`,
		userID, viewerID).Scan(&s.Timelines, &s.Events, &s.ApprovedContributions, &s.Followers, &s.Following)
	return s, translate(err)
}

type sessions struct{ *db }

func (r sessions) Create(ctx context.Context, s domain.Session) error {
	_, err := r.q(ctx).Exec(ctx, `INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES ($1,$2,$3,$4)`,
		s.TokenHash, s.UserID, s.CreatedAt, s.ExpiresAt)
	return translate(err)
}

func (r sessions) UserID(ctx context.Context, hash string, now time.Time) (string, error) {
	var id string
	err := r.q(ctx).QueryRow(ctx, `SELECT user_id FROM sessions WHERE token_hash=$1 AND expires_at > $2`, hash, now).Scan(&id)
	return id, translate(err)
}
