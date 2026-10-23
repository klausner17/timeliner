package usecase

import (
	"context"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

type ListEventsInput struct {
	ActorID string
	Slug    string
	From    *int
	To      *int
}

type ListEvents struct{ d *Deps }

func NewListEvents(d *Deps) *ListEvents { return &ListEvents{d} }

func (uc *ListEvents) Execute(ctx context.Context, in ListEventsInput) ([]EventView, error) {
	t, err := visibleTimeline(ctx, uc.d, in.Slug, in.ActorID)
	if err != nil {
		return nil, err
	}
	if in.From != nil && in.To != nil && *in.From > *in.To {
		return nil, domain.Invalid(map[string]string{"to": "O ano final precisa ser depois do inicial."})
	}
	events, err := uc.d.Events.ListByTimeline(ctx, t.ID, in.From, in.To)
	if err != nil {
		return nil, err
	}
	return eventViews(ctx, uc.d, events)
}

type CreateEventInput struct {
	ActorID string
	Slug    string
	Event   domain.EventData
}

// CreateEvent adiciona um evento direto. Só o dono; os demais mandam sugestões.
type CreateEvent struct{ d *Deps }

func NewCreateEvent(d *Deps) *CreateEvent { return &CreateEvent{d} }

func (uc *CreateEvent) Execute(ctx context.Context, in CreateEventInput) (EventView, error) {
	t, err := ownedTimeline(ctx, uc.d, in.Slug, in.ActorID)
	if err != nil {
		return EventView{}, err
	}
	data := normalizeEvent(in.Event)
	f := fieldErrors{}
	validateEvent(f, "", data, t.Eras)
	if err := f.err(); err != nil {
		return EventView{}, err
	}
	e := domain.Event{EventData: data, ID: uc.d.IDs.NewID("ev"), TimelineID: t.ID, AuthorID: in.ActorID, CreatedAt: uc.d.Clock.Now()}
	err = uc.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.d.Events.Create(ctx, &e); err != nil {
			return err
		}
		return touch(ctx, uc.d, t)
	})
	if err != nil {
		return EventView{}, err
	}
	return eventView(ctx, uc.d, &e)
}

// touch marca a timeline como atualizada agora.
func touch(ctx context.Context, d *Deps, t *domain.Timeline) error {
	t.UpdatedAt = d.Clock.Now()
	return d.Timelines.Update(ctx, t)
}

type EventPatch struct {
	Year        *int
	EndYear     **int
	Date        **string
	Title       *string
	Description *string
	SourceURL   **string
	Era         **string
}

type UpdateEventInput struct {
	ActorID string
	Slug    string
	EventID string
	Patch   EventPatch
}

type UpdateEvent struct{ d *Deps }

func NewUpdateEvent(d *Deps) *UpdateEvent { return &UpdateEvent{d} }

func (uc *UpdateEvent) Execute(ctx context.Context, in UpdateEventInput) (EventView, error) {
	t, e, err := ownedEvent(ctx, uc.d, in.Slug, in.EventID, in.ActorID)
	if err != nil {
		return EventView{}, err
	}
	p := in.Patch
	if p.Year != nil {
		e.Year = *p.Year
	}
	if p.EndYear != nil {
		e.EndYear = *p.EndYear
	}
	if p.Date != nil {
		e.Date = *p.Date
	}
	if p.Title != nil {
		e.Title = *p.Title
	}
	if p.Description != nil {
		e.Description = *p.Description
	}
	if p.SourceURL != nil {
		e.SourceURL = *p.SourceURL
	}
	if p.Era != nil {
		e.Era = *p.Era
	}
	e.EventData = normalizeEvent(e.EventData)
	f := fieldErrors{}
	validateEvent(f, "", e.EventData, t.Eras)
	if err := f.err(); err != nil {
		return EventView{}, err
	}
	err = uc.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.d.Events.Update(ctx, e); err != nil {
			return err
		}
		return touch(ctx, uc.d, t)
	})
	if err != nil {
		return EventView{}, err
	}
	return eventView(ctx, uc.d, e)
}

type EventRefInput struct {
	ActorID string
	Slug    string
	EventID string
}

type DeleteEvent struct{ d *Deps }

func NewDeleteEvent(d *Deps) *DeleteEvent { return &DeleteEvent{d} }

func (uc *DeleteEvent) Execute(ctx context.Context, in EventRefInput) (Empty, error) {
	t, e, err := ownedEvent(ctx, uc.d, in.Slug, in.EventID, in.ActorID)
	if err != nil {
		return Empty{}, err
	}
	return Empty{}, uc.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.d.Events.Delete(ctx, e.ID); err != nil {
			return err
		}
		return touch(ctx, uc.d, t)
	})
}

func ownedEvent(ctx context.Context, d *Deps, slug, eventID, actorID string) (*domain.Timeline, *domain.Event, error) {
	t, err := ownedTimeline(ctx, d, slug, actorID)
	if err != nil {
		return nil, nil, err
	}
	e, err := eventOf(ctx, d, t, eventID)
	return t, e, err
}

// eventOf busca o evento e confere que ele é da timeline.
func eventOf(ctx context.Context, d *Deps, t *domain.Timeline, eventID string) (*domain.Event, error) {
	e, err := d.Events.ByID(ctx, eventID)
	if err == nil && e.TimelineID != t.ID {
		err = domain.ErrNotFound
	}
	if err != nil {
		return nil, notFound(err, "Evento não encontrado.")
	}
	return e, nil
}
