package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

// Limites para timelines grandes: ninguém recebe todos os eventos de uma vez.
const (
	DefaultPageSize  = 100
	MaxPageSize      = 500
	DefaultBuckets   = 50
	MaxBuckets       = 200
	DefaultMaxEvents = 150
	MaxEvents        = 500
	// MaxImport é quantos eventos cabem na criação de uma timeline.
	MaxImport   = 500
	maxOverlaps = 200
	sampleSize  = 3
)

type EventPage struct {
	Items []EventView
	// NextCursor é nil na última página.
	NextCursor *string
}

// eventPage busca uma página e monta o cursor da próxima, pedindo um evento a mais
// para saber se ela existe.
func eventPage(ctx context.Context, d *Deps, q EventQuery) (EventPage, error) {
	want := q.Limit
	q.Limit++
	events, err := d.Events.ListByTimeline(ctx, q)
	if err != nil {
		return EventPage{}, err
	}
	var next *string
	if len(events) > want {
		events = events[:want]
		c := encodeCursor(domain.CursorOf(events[want-1]))
		next = &c
	}
	evs, err := eventViews(ctx, d, events)
	if err != nil {
		return EventPage{}, err
	}
	return EventPage{Items: evs, NextCursor: next}, nil
}

type cursorJSON struct {
	Year      int       `json:"y"`
	Month     int       `json:"m"`
	Day       int       `json:"d"`
	Time      string    `json:"t"`
	EndYear   int       `json:"e"`
	CreatedAt time.Time `json:"c"`
	ID        string    `json:"i"`
}

func encodeCursor(c domain.EventCursor) string {
	b, _ := json.Marshal(cursorJSON(c))
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (*domain.EventCursor, bool) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	var c cursorJSON
	if err != nil || json.Unmarshal(b, &c) != nil || c.ID == "" {
		return nil, false
	}
	dc := domain.EventCursor(c)
	return &dc, true
}

type WindowMode string

const (
	// WindowEvents: o intervalo tem poucos eventos e eles vêm todos.
	WindowEvents WindowMode = "eventos"
	// WindowBuckets: eventos demais, vêm agrupados por faixas de anos.
	WindowBuckets WindowMode = "faixas"
)

type BucketView struct {
	From   int
	To     int
	Count  int
	Sample []EventView
}

// EventWindow é o que cabe num trecho do eixo: os eventos, se forem até MaxEvents, ou
// as faixas com contagem e exemplos. Quem decide é a quantidade, não o tamanho do trecho.
type EventWindow struct {
	From    int
	To      int
	Total   int
	Mode    WindowMode
	Events  []EventView
	Buckets []BucketView
}

type WindowOptions struct {
	// Buckets é em quantas faixas dividir o trecho quando houver eventos demais.
	Buckets int
	// MaxEvents é até quantos eventos vêm um a um.
	MaxEvents int
}

func (o *WindowOptions) validate(f fieldErrors) {
	if o.Buckets == 0 {
		o.Buckets = DefaultBuckets
	}
	if o.MaxEvents == 0 {
		o.MaxEvents = DefaultMaxEvents
	}
	f.check(o.Buckets >= 1 && o.Buckets <= MaxBuckets, "buckets", "Use de 1 a 200 faixas.")
	f.check(o.MaxEvents >= 1 && o.MaxEvents <= MaxEvents, "maxEvents", "Use de 1 a 500 eventos.")
}

func eventWindow(ctx context.Context, d *Deps, timelineID string, from, to int, o WindowOptions) (EventWindow, error) {
	w := EventWindow{From: from, To: to, Mode: WindowEvents, Events: []EventView{}, Buckets: []BucketView{}}
	events, err := d.Events.ListByTimeline(ctx, EventQuery{TimelineID: timelineID, From: &from, To: &to, Limit: o.MaxEvents + 1})
	if err != nil {
		return EventWindow{}, err
	}
	if len(events) <= o.MaxEvents {
		w.Total = len(events)
		w.Events, err = eventViews(ctx, d, events)
		return w, err
	}
	buckets, err := d.Events.Summary(ctx, SummaryQuery{TimelineID: timelineID, From: from, To: to, Buckets: min(o.Buckets, to-from+1), Sample: sampleSize})
	if err != nil {
		return EventWindow{}, err
	}
	samples := []domain.Event{}
	for _, b := range buckets {
		samples = append(samples, b.Sample...)
		w.Total += b.Count
	}
	views, err := eventViews(ctx, d, samples)
	if err != nil {
		return EventWindow{}, err
	}
	w.Mode = WindowBuckets
	w.Buckets = make([]BucketView, len(buckets))
	for i, b := range buckets {
		w.Buckets[i] = BucketView{From: b.From, To: b.To, Count: b.Count, Sample: views[:len(b.Sample)]}
		views = views[len(b.Sample):]
	}
	return w, nil
}

type SummarizeEventsInput struct {
	ActorID string
	Slug    string
	// From e To vazios usam o primeiro e o último ano da timeline.
	From *int
	To   *int
	WindowOptions
}

// SummarizeEvents é a visão do eixo (modo horizontal, minimapa): poucos eventos vêm
// todos, muitos vêm em faixas de anos.
type SummarizeEvents struct{ d *Deps }

func NewSummarizeEvents(d *Deps) *SummarizeEvents { return &SummarizeEvents{d} }

func (uc *SummarizeEvents) Execute(ctx context.Context, in SummarizeEventsInput) (EventWindow, error) {
	t, err := visibleTimeline(ctx, uc.d, in.Slug, in.ActorID)
	if err != nil {
		return EventWindow{}, err
	}
	f := fieldErrors{}
	in.WindowOptions.validate(f)
	f.check(in.From == nil || in.To == nil || *in.From <= *in.To, "to", "O ano final precisa ser depois do inicial.")
	if err := f.err(); err != nil {
		return EventWindow{}, err
	}
	stats, err := uc.d.Timelines.Stats(ctx, []string{t.ID}, in.ActorID)
	if err != nil {
		return EventWindow{}, err
	}
	s := stats[t.ID]
	from, to := 0, 0
	if s.StartYear != nil {
		from, to = *s.StartYear, *s.EndYear
	}
	if in.From != nil {
		from = *in.From
	}
	if in.To != nil {
		to = *in.To
	}
	if to < from {
		to = from
	}
	return eventWindow(ctx, uc.d, t.ID, from, to, in.WindowOptions)
}
