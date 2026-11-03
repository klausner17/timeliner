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

func (r suggestions) List(ctx context.Context, f usecase.SuggestionFilter) ([]domain.Suggestion, int, error) {
	where := []string{"true"}
	args := []any{}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if f.TimelineID != "" {
		where = append(where, "timeline_id="+arg(f.TimelineID))
	}
	if f.AuthorID != "" {
		where = append(where, "author_id="+arg(f.AuthorID))
	}
	if f.Status != "" {
		where = append(where, "status="+arg(string(f.Status)))
	}
	if f.ViewerID != "" || f.OnlyVisible {
		v := arg(f.ViewerID)
		where = append(where, `EXISTS (SELECT 1 FROM timelines t WHERE t.id=suggestions.timeline_id
			AND ((NOT t.draft AND t.visibility<>'privada') OR t.owner_id=`+v+`))`)
	}
	cond := strings.Join(where, " AND ")
	var total int
	if err := r.q(ctx).QueryRow(ctx, `SELECT count(*) FROM suggestions WHERE `+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.q(ctx).Query(ctx, `SELECT `+suggestionCols+` FROM suggestions WHERE `+cond+
		` ORDER BY created_at DESC, id DESC LIMIT `+arg(f.Limit)+` OFFSET `+arg(f.Offset), args...)
	if err != nil {
		return nil, 0, err
	}
	ss, err := collect(rows, scanSuggestion)
	return ss, total, err
}

func (r suggestions) ShiftYears(ctx context.Context, timelineIDs []string, shifts []domain.YearShift) error {
	rows, err := r.q(ctx).Query(ctx, `SELECT `+suggestionCols+` FROM suggestions WHERE timeline_id = ANY($1)`, timelineIDs)
	if err != nil {
		return err
	}
	ss, err := collect(rows, scanSuggestion)
	if err != nil {
		return err
	}
	for _, s := range ss {
		s.Proposed = shiftData(s.Proposed, shifts)
		if s.Previous != nil {
			s.Previous.EventData = shiftData(s.Previous.EventData, shifts)
		}
		if err := r.Update(ctx, &s); err != nil {
			return err
		}
	}
	return nil
}

func shiftData(e domain.EventData, shifts []domain.YearShift) domain.EventData {
	e.Year = domain.ShiftYear(e.Year, shifts)
	if e.EndYear != nil {
		end := domain.ShiftYear(*e.EndYear, shifts)
		e.EndYear = &end
	}
	return e
}
