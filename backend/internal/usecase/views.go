package usecase

import (
	"context"
	"errors"
	"slices"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

// As views são o Output dos casos de uso: entidades já com o que a tela precisa
// (handles, números). O controller só converte para JSON.

type UserView struct {
	User  domain.User
	Stats domain.UserStats
}

type TimelineView struct {
	Timeline domain.Timeline
	Owner    domain.User
	Stats    domain.TimelineStats
	// Points são as posições (0 a 100) dos eventos no eixo, para o mini trilho.
	Points []float64
}

type EventView struct {
	Event        domain.Event
	AuthorHandle string
}

type TimelineDetailView struct {
	TimelineView
	Events []EventView
}

type SuggestionView struct {
	Suggestion   domain.Suggestion
	Timeline     domain.Timeline
	AuthorHandle string
	Previous     *EventView
}

const maxPoints = 24

// points espalha os anos no eixo de 0 a 100, sem repetir e com no máximo maxPoints.
func points(years []int, start, end *int) []float64 {
	out := []float64{}
	if len(years) == 0 || start == nil || end == nil {
		return out
	}
	span := float64(*end - *start)
	for _, y := range years {
		p := 50.0
		if span > 0 {
			p = float64(y-*start) / span * 100
		}
		p = float64(int(p*10+0.5)) / 10
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	if len(out) <= maxPoints {
		return out
	}
	sampled := make([]float64, 0, maxPoints)
	for i := range maxPoints {
		sampled = append(sampled, out[i*(len(out)-1)/(maxPoints-1)])
	}
	return sampled
}

func timelineViews(ctx context.Context, d *Deps, ts []domain.Timeline, viewerID string) ([]TimelineView, error) {
	ids := make([]string, len(ts))
	ownerIDs := make([]string, len(ts))
	for i, t := range ts {
		ids[i], ownerIDs[i] = t.ID, t.OwnerID
	}
	stats, err := d.Timelines.Stats(ctx, ids, viewerID)
	if err != nil {
		return nil, err
	}
	owners, err := d.Users.ByIDs(ctx, ownerIDs)
	if err != nil {
		return nil, err
	}
	out := make([]TimelineView, len(ts))
	for i, t := range ts {
		s := stats[t.ID]
		out[i] = TimelineView{Timeline: t, Owner: owners[t.OwnerID], Stats: s, Points: points(s.Years, s.StartYear, s.EndYear)}
	}
	return out, nil
}

func timelineView(ctx context.Context, d *Deps, t *domain.Timeline, viewerID string) (TimelineView, error) {
	vs, err := timelineViews(ctx, d, []domain.Timeline{*t}, viewerID)
	if err != nil {
		return TimelineView{}, err
	}
	return vs[0], nil
}

func timelineDetail(ctx context.Context, d *Deps, t *domain.Timeline, viewerID string) (TimelineDetailView, error) {
	tv, err := timelineView(ctx, d, t, viewerID)
	if err != nil {
		return TimelineDetailView{}, err
	}
	events, err := d.Events.ListByTimeline(ctx, t.ID, nil, nil)
	if err != nil {
		return TimelineDetailView{}, err
	}
	evs, err := eventViews(ctx, d, events)
	if err != nil {
		return TimelineDetailView{}, err
	}
	return TimelineDetailView{TimelineView: tv, Events: evs}, nil
}

func eventViews(ctx context.Context, d *Deps, events []domain.Event) ([]EventView, error) {
	ids := make([]string, len(events))
	for i, e := range events {
		ids[i] = e.AuthorID
	}
	users, err := d.Users.ByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]EventView, len(events))
	for i, e := range events {
		out[i] = EventView{Event: e, AuthorHandle: users[e.AuthorID].Handle}
	}
	return out, nil
}

func eventView(ctx context.Context, d *Deps, e *domain.Event) (EventView, error) {
	vs, err := eventViews(ctx, d, []domain.Event{*e})
	if err != nil {
		return EventView{}, err
	}
	return vs[0], nil
}

// suggestionViews completa as sugestões com timeline e handles. Sugestões de timelines
// que o viewer não pode ver ficam de fora.
func suggestionViews(ctx context.Context, d *Deps, ss []domain.Suggestion, viewerID string) ([]SuggestionView, error) {
	userIDs := []string{}
	timelines := map[string]*domain.Timeline{}
	for _, s := range ss {
		userIDs = append(userIDs, s.AuthorID)
		if s.Previous != nil {
			userIDs = append(userIDs, s.Previous.AuthorID)
		}
		if _, ok := timelines[s.TimelineID]; !ok {
			t, err := d.Timelines.ByID(ctx, s.TimelineID)
			if err != nil && !errors.Is(err, domain.ErrNotFound) {
				return nil, err
			}
			timelines[s.TimelineID] = t
		}
	}
	users, err := d.Users.ByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := make([]SuggestionView, 0, len(ss))
	for _, s := range ss {
		t := timelines[s.TimelineID]
		if t == nil || !t.VisibleTo(viewerID) {
			continue
		}
		v := SuggestionView{Suggestion: s, Timeline: *t, AuthorHandle: users[s.AuthorID].Handle}
		if s.Previous != nil {
			v.Previous = &EventView{Event: *s.Previous, AuthorHandle: users[s.Previous.AuthorID].Handle}
		}
		out = append(out, v)
	}
	return out, nil
}

func suggestionView(ctx context.Context, d *Deps, s *domain.Suggestion, viewerID string) (SuggestionView, error) {
	vs, err := suggestionViews(ctx, d, []domain.Suggestion{*s}, viewerID)
	if err != nil {
		return SuggestionView{}, err
	}
	if len(vs) == 0 {
		return SuggestionView{}, domain.NotFound("Sugestão não encontrada.")
	}
	return vs[0], nil
}

// visibleTimeline busca pelo slug e esconde (404) o que o viewer não pode ver.
func visibleTimeline(ctx context.Context, d *Deps, slug, viewerID string) (*domain.Timeline, error) {
	t, err := d.Timelines.BySlug(ctx, slug)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && !t.VisibleTo(viewerID)) {
		return nil, domain.NotFound("Timeline não encontrada.")
	}
	return t, err
}

// ownedTimeline é visibleTimeline mais a exigência de ser o dono.
func ownedTimeline(ctx context.Context, d *Deps, slug, actorID string) (*domain.Timeline, error) {
	if actorID == "" {
		return nil, domain.Unauthorized("Entre na sua conta para continuar.")
	}
	t, err := visibleTimeline(ctx, d, slug, actorID)
	if err != nil {
		return nil, err
	}
	if t.OwnerID != actorID {
		return nil, domain.Forbidden("Só o dono da timeline pode fazer isso.")
	}
	return t, nil
}

func requireActor(actorID string) error {
	if actorID == "" {
		return domain.Unauthorized("Entre na sua conta para continuar.")
	}
	return nil
}

// notFound troca o ErrNotFound do repository pelo erro de negócio com a mensagem dada.
func notFound(err error, msg string) error {
	if errors.Is(err, domain.ErrNotFound) {
		return domain.NotFound(msg)
	}
	return err
}
