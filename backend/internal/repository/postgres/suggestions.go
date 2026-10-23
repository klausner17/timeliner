package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

type suggestions struct{ *db }

const suggestionCols = `id, timeline_id, kind, status, author_id, event_id, proposed, note, feedback,
	applied_event_id, previous, created_at, reviewed_at`

func scanSuggestion(row pgx.Row) (*domain.Suggestion, error) {
	var s domain.Suggestion
	var proposed, previous []byte
	err := row.Scan(&s.ID, &s.TimelineID, &s.Kind, &s.Status, &s.AuthorID, &s.EventID, &proposed, &s.Note,
		&s.Feedback, &s.AppliedEventID, &previous, &s.CreatedAt, &s.ReviewedAt)
	if err != nil {
		return nil, translate(err)
	}
	var p eventJSON
	if err := json.Unmarshal(proposed, &p); err != nil {
		return nil, err
	}
	s.Proposed = p.data()
	if s.Previous, err = unmarshalEvent(previous); err != nil {
		return nil, err
	}
	return &s, nil
}

func suggestionArgs(s *domain.Suggestion) ([]any, error) {
	proposed, err := json.Marshal(dataToJSON(s.Proposed))
	if err != nil {
		return nil, err
	}
	previous, err := marshalEvent(s.Previous)
	if err != nil {
		return nil, err
	}
	return []any{s.ID, s.TimelineID, s.Kind, s.Status, s.AuthorID, s.EventID, proposed, s.Note, s.Feedback,
		s.AppliedEventID, previous, s.CreatedAt, s.ReviewedAt}, nil
}

func (r suggestions) Create(ctx context.Context, s *domain.Suggestion) error {
	args, err := suggestionArgs(s)
	if err != nil {
		return err
	}
	_, err = r.q(ctx).Exec(ctx, `INSERT INTO suggestions (`+suggestionCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, args...)
	return translate(err)
}

func (r suggestions) Update(ctx context.Context, s *domain.Suggestion) error {
	args, err := suggestionArgs(s)
	if err != nil {
		return err
	}
	return mustAffect(r.q(ctx).Exec(ctx, `UPDATE suggestions SET timeline_id=$2, kind=$3, status=$4, author_id=$5,
		event_id=$6, proposed=$7, note=$8, feedback=$9, applied_event_id=$10, previous=$11, created_at=$12,
		reviewed_at=$13 WHERE id=$1`, args...))
}

func (r suggestions) ByID(ctx context.Context, id string) (*domain.Suggestion, error) {
	return scanSuggestion(r.q(ctx).QueryRow(ctx, `SELECT `+suggestionCols+` FROM suggestions WHERE id=$1`, id))
}

func (r suggestions) ListByTimeline(ctx context.Context, timelineID string, status domain.SuggestionStatus) ([]domain.Suggestion, error) {
	rows, err := r.q(ctx).Query(ctx, `SELECT `+suggestionCols+` FROM suggestions
		WHERE timeline_id=$1 AND ($2 = '' OR status=$2) ORDER BY created_at DESC, id DESC`, timelineID, string(status))
	if err != nil {
		return nil, err
	}
	return collect(rows, scanSuggestion)
}

func (r suggestions) ListByAuthor(ctx context.Context, authorID string) ([]domain.Suggestion, error) {
	rows, err := r.q(ctx).Query(ctx, `SELECT `+suggestionCols+` FROM suggestions
		WHERE author_id=$1 ORDER BY created_at DESC, id DESC`, authorID)
	if err != nil {
		return nil, err
	}
	return collect(rows, scanSuggestion)
}
