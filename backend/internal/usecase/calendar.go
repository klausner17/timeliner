package usecase

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

// MaxCalendarEras é quantas eras de contagem um calendário pode ter.
const MaxCalendarEras = 30

type CalendarFields struct {
	Name        *string
	Description *string
	Shared      *bool
	// Eras substitui a lista inteira. Era com ID é uma que já existia (pode mudar nome,
	// sigla e duração); sem ID é nova.
	Eras *[]domain.CalendarEra
}

// apply copia os campos preenchidos para c e valida. newID gera o ID das eras novas.
func (in CalendarFields) apply(c *domain.Calendar, newID func() string) error {
	f := fieldErrors{}
	if in.Name != nil {
		c.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		c.Description = strings.TrimSpace(*in.Description)
	}
	if in.Shared != nil {
		c.Shared = *in.Shared
	}
	if in.Eras != nil {
		known := map[string]bool{}
		for _, e := range c.Eras {
			known[e.ID] = true
		}
		eras := make([]domain.CalendarEra, len(*in.Eras))
		for i, e := range *in.Eras {
			e.Name, e.Abbr, e.ID = strings.TrimSpace(e.Name), strings.TrimSpace(e.Abbr), strings.TrimSpace(e.ID)
			if e.ID == "" {
				e.ID = newID()
			} else {
				f.check(known[e.ID], fmt.Sprintf("eras[%d].id", i), "Era não existe neste calendário.")
			}
			eras[i] = e
		}
		c.Eras = eras
	}
	f.check(length(c.Name, 1, 80), "name", "O nome precisa ter de 1 a 80 caracteres.")
	f.check(length(c.Description, 0, 2000), "description", "A descrição pode ter até 2000 caracteres.")
	f.check(len(c.Eras) >= 1, "eras", "O calendário precisa de pelo menos uma era.")
	f.check(len(c.Eras) <= MaxCalendarEras, "eras", fmt.Sprintf("No máximo %d eras.", MaxCalendarEras))
	names, ids := map[string]bool{}, map[string]bool{}
	for i, e := range c.Eras {
		p := fmt.Sprintf("eras[%d].", i)
		f.check(length(e.Name, 1, 60), p+"name", "O nome da era precisa ter de 1 a 60 caracteres.")
		f.check(!names[strings.ToLower(e.Name)], p+"name", "Já existe uma era com esse nome.")
		f.check(length(e.Abbr, 1, 12), p+"abbr", "A sigla precisa ter de 1 a 12 caracteres.")
		f.check(!ids[e.ID], p+"id", "Era repetida.")
		last := i == len(c.Eras)-1
		if e.Years == nil {
			f.check(last, p+"years", "Só a última era pode ficar sem duração.")
		} else {
			f.check(*e.Years >= 1 && *e.Years <= domain.OpenEraYears, p+"years",
				fmt.Sprintf("A duração precisa ser de 1 a %d anos.", domain.OpenEraYears))
		}
		names[strings.ToLower(e.Name)], ids[e.ID] = true, true
	}
	return f.err()
}

// CalendarView é o calendário com o handle do dono.
type CalendarView struct {
	Calendar    domain.Calendar
	OwnerHandle string
}

func calendarViews(ctx context.Context, d *Deps, cs []domain.Calendar) ([]CalendarView, error) {
	ids := make([]string, len(cs))
	for i, c := range cs {
		ids[i] = c.OwnerID
	}
	users, err := d.Users.ByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]CalendarView, len(cs))
	for i, c := range cs {
		out[i] = CalendarView{Calendar: c, OwnerHandle: users[c.OwnerID].Handle}
	}
	return out, nil
}

func calendarView(ctx context.Context, d *Deps, c *domain.Calendar) (CalendarView, error) {
	vs, err := calendarViews(ctx, d, []domain.Calendar{*c})
	if err != nil {
		return CalendarView{}, err
	}
	return vs[0], nil
}

type CreateCalendarInput struct {
	ActorID string
	CalendarFields
}

type CreateCalendar struct{ d *Deps }

func NewCreateCalendar(d *Deps) *CreateCalendar { return &CreateCalendar{d} }

func (uc *CreateCalendar) Execute(ctx context.Context, in CreateCalendarInput) (CalendarView, error) {
	if err := requireActor(in.ActorID); err != nil {
		return CalendarView{}, err
	}
	now := uc.d.Clock.Now()
	c := domain.Calendar{ID: uc.d.IDs.NewID("cal"), OwnerID: in.ActorID, CreatedAt: now, UpdatedAt: now}
	if err := in.CalendarFields.apply(&c, func() string { return uc.d.IDs.NewID("era") }); err != nil {
		return CalendarView{}, err
	}
	if err := uc.d.Calendars.Create(ctx, &c); err != nil {
		return CalendarView{}, err
	}
	return calendarView(ctx, uc.d, &c)
}

type ListCalendarsInput struct {
	ActorID string
	Query   string
}

// ListCalendars devolve os calendários que o usuário pode usar: os dele e os compartilhados.
type ListCalendars struct{ d *Deps }

func NewListCalendars(d *Deps) *ListCalendars { return &ListCalendars{d} }

func (uc *ListCalendars) Execute(ctx context.Context, in ListCalendarsInput) ([]CalendarView, error) {
	cs, err := uc.d.Calendars.Usable(ctx, in.ActorID, strings.TrimSpace(in.Query))
	if err != nil {
		return nil, err
	}
	return calendarViews(ctx, uc.d, cs)
}

type CalendarRefInput struct {
	ActorID    string
	CalendarID string
}

type GetCalendar struct{ d *Deps }

func NewGetCalendar(d *Deps) *GetCalendar { return &GetCalendar{d} }

func (uc *GetCalendar) Execute(ctx context.Context, in CalendarRefInput) (CalendarView, error) {
	c, err := usableCalendar(ctx, uc.d, in.CalendarID, in.ActorID)
	if err != nil {
		return CalendarView{}, err
	}
	return calendarView(ctx, uc.d, c)
}

// usableCalendar busca o calendário e esconde (404) o que o usuário não pode usar.
func usableCalendar(ctx context.Context, d *Deps, id, actorID string) (*domain.Calendar, error) {
	c, err := d.Calendars.ByID(ctx, id)
	if err == nil && !c.Shared && c.OwnerID != actorID {
		err = domain.ErrNotFound
	}
	if err != nil {
		return nil, notFound(err, "Calendário não encontrado.")
	}
	return c, nil
}

func ownedCalendar(ctx context.Context, d *Deps, id, actorID string) (*domain.Calendar, error) {
	if err := requireActor(actorID); err != nil {
		return nil, err
	}
	c, err := usableCalendar(ctx, d, id, actorID)
	if err != nil {
		return nil, err
	}
	if c.OwnerID != actorID {
		return nil, domain.Forbidden("Só o dono do calendário pode fazer isso.")
	}
	return c, nil
}

type UpdateCalendarInput struct {
	ActorID    string
	CalendarID string
	CalendarFields
}

// UpdateCalendar muda o calendário. Quando uma era muda de duração (ou entra uma era nova
// no meio), as eras seguintes andam no eixo, e os eventos, períodos e sugestões de todas
// as timelines que usam o calendário andam junto: "T.A. 3019" continua "T.A. 3019".
type UpdateCalendar struct{ d *Deps }

func NewUpdateCalendar(d *Deps) *UpdateCalendar { return &UpdateCalendar{d} }

func (uc *UpdateCalendar) Execute(ctx context.Context, in UpdateCalendarInput) (CalendarView, error) {
	c, err := ownedCalendar(ctx, uc.d, in.CalendarID, in.ActorID)
	if err != nil {
		return CalendarView{}, err
	}
	before := *c
	before.Eras = slices.Clone(c.Eras)
	if err := in.CalendarFields.apply(c, func() string { return uc.d.IDs.NewID("era") }); err != nil {
		return CalendarView{}, err
	}
	c.UpdatedAt = uc.d.Clock.Now()
	err = uc.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if in.Eras != nil {
			if err := moveCalendarContent(ctx, uc.d, &before, c); err != nil {
				return err
			}
		}
		return uc.d.Calendars.Update(ctx, c)
	})
	if err != nil {
		return CalendarView{}, err
	}
	return calendarView(ctx, uc.d, c)
}

// moveCalendarContent confere que tudo que está nas timelines do calendário cabe nas eras
// novas e desloca os anos para as eras continuarem com a mesma contagem.
func moveCalendarContent(ctx context.Context, d *Deps, before, after *domain.Calendar) error {
	ts, err := d.Timelines.ByCalendar(ctx, before.ID)
	if err != nil || len(ts) == 0 {
		return err
	}
	ids := make([]string, len(ts))
	for i, t := range ts {
		ids[i] = t.ID
	}
	newSpans := map[string]domain.EraSpan{}
	order := map[string]int{}
	for i, s := range after.Spans() {
		newSpans[s.ID] = s
		order[s.ID] = i
	}
	f := fieldErrors{}
	shifts := []domain.YearShift{}
	lastOrder := -1
	for i, old := range before.Spans() {
		name := before.Eras[i].Name
		latest, err := latestUsedYear(ctx, d, ts, ids, old.Start, old.End)
		if err != nil {
			return err
		}
		next, kept := newSpans[old.ID]
		if !kept {
			f.check(latest == nil, "eras", fmt.Sprintf("A era %q tem eventos ou períodos; mova-os antes de apagar a era.", name))
			continue
		}
		f.check(order[old.ID] > lastOrder, "eras", "Não dá para trocar a ordem das eras que já existem.")
		lastOrder = order[old.ID]
		if latest != nil && *latest-old.Start > next.End-next.Start {
			f.add("eras", fmt.Sprintf("A era %q ficaria curta demais: ela tem conteúdo até o ano %d.", name, *latest-old.Start+1))
		}
		if delta := next.Start - old.Start; delta != 0 {
			shifts = append(shifts, domain.YearShift{From: old.Start, To: old.End, Delta: delta})
		}
	}
	if err := f.err(); err != nil {
		return err
	}
	if len(shifts) == 0 {
		return nil
	}
	if err := d.Events.ShiftYears(ctx, ids, shifts); err != nil {
		return err
	}
	if err := d.Suggestions.ShiftYears(ctx, ids, shifts); err != nil {
		return err
	}
	for i := range ts {
		t := &ts[i]
		for j := range t.Eras {
			e := &t.Eras[j]
			e.Start = domain.ShiftYear(e.Start, shifts)
			if e.End != nil {
				end := domain.ShiftYear(*e.End, shifts)
				e.End = &end
			}
		}
		if err := d.Timelines.Update(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

// latestUsedYear é o maior ano em [from, to] usado por evento ou período das timelines.
func latestUsedYear(ctx context.Context, d *Deps, ts []domain.Timeline, ids []string, from, to int) (*int, error) {
	latest, err := d.Events.LatestYearIn(ctx, ids, from, to)
	if err != nil {
		return nil, err
	}
	use := func(y int) {
		if y >= from && y <= to && (latest == nil || y > *latest) {
			latest = &y
		}
	}
	for _, t := range ts {
		for _, e := range t.Eras {
			use(e.Start)
			if e.End != nil {
				use(*e.End)
			}
		}
	}
	return latest, nil
}

type DeleteCalendar struct{ d *Deps }

func NewDeleteCalendar(d *Deps) *DeleteCalendar { return &DeleteCalendar{d} }

func (uc *DeleteCalendar) Execute(ctx context.Context, in CalendarRefInput) (Empty, error) {
	c, err := ownedCalendar(ctx, uc.d, in.CalendarID, in.ActorID)
	if err != nil {
		return Empty{}, err
	}
	used, err := uc.d.Calendars.InUse(ctx, c.ID)
	if err != nil {
		return Empty{}, err
	}
	if used {
		return Empty{}, domain.Conflict("Este calendário ainda é usado por timelines.")
	}
	return Empty{}, uc.d.Calendars.Delete(ctx, c.ID)
}

// yearRules são os limites de ano de uma timeline: os do calendário gregoriano ou os do
// calendário ficcional, que também não tem mês nem dia.
type yearRules struct {
	Lo, Hi   int
	YearOnly bool
}

func rulesFor(c *domain.Calendar) yearRules {
	if c == nil {
		return yearRules{Lo: minYear, Hi: maxYear}
	}
	lo, hi := c.Bounds()
	return yearRules{Lo: lo, Hi: hi, YearOnly: true}
}

// calendarOf devolve o calendário da timeline, ou nil no gregoriano.
func calendarOf(ctx context.Context, d *Deps, t *domain.Timeline) (*domain.Calendar, error) {
	if t.CalendarID == nil {
		return nil, nil
	}
	return d.Calendars.ByID(ctx, *t.CalendarID)
}

func rulesOf(ctx context.Context, d *Deps, t *domain.Timeline) (yearRules, error) {
	c, err := calendarOf(ctx, d, t)
	return rulesFor(c), err
}
