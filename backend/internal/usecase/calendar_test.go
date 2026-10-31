package usecase_test

import (
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func middleEarth() []domain.CalendarEra {
	return []domain.CalendarEra{
		{Name: "Primeira Era", Abbr: "P.E.", Years: ptr(590)},
		{Name: "Segunda Era", Abbr: "S.E.", Years: ptr(3440)},
		{Name: "Terceira Era", Abbr: "T.A.", Years: ptr(3021)},
		{Name: "Quarta Era", Abbr: "Q.E."},
	}
}

func createCalendar(t *testing.T, d *usecase.Deps, owner string, shared bool) usecase.CalendarView {
	t.Helper()
	eras := middleEarth()
	return must[usecase.CalendarView](t)(usecase.NewCreateCalendar(d).Execute(ctx, usecase.CreateCalendarInput{
		ActorID: owner, CalendarFields: usecase.CalendarFields{Name: ptr("Terra-média"), Shared: &shared, Eras: &eras},
	}))
}

func createFictional(t *testing.T, d *usecase.Deps, owner, title, calendarID string, events ...domain.EventData) (usecase.TimelineDetailView, error) {
	t.Helper()
	return usecase.NewCreateTimeline(d).Execute(ctx, usecase.CreateTimelineInput{
		ActorID: owner, CalendarID: calendarID,
		TimelineFields: usecase.TimelineFields{Title: &title, Category: ptr("Cultura")},
		Events:         events,
	})
}

// T.A. 3019 no calendário de middleEarth (antes de qualquer mudança).
const ta3019 = 590 + 3440 + 3019

func TestCalendarValidation(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	create := usecase.NewCreateCalendar(d)
	for name, eras := range map[string][]domain.CalendarEra{
		"sem eras":          {},
		"aberta no meio":    {{Name: "A", Abbr: "A"}, {Name: "B", Abbr: "B", Years: ptr(10)}},
		"nome repetido":     {{Name: "A", Abbr: "A", Years: ptr(10)}, {Name: "a", Abbr: "B"}},
		"sem sigla":         {{Name: "A", Abbr: " "}},
		"duração zero":      {{Name: "A", Abbr: "A", Years: ptr(0)}},
		"id que não existe": {{ID: "era_x", Name: "A", Abbr: "A"}},
	} {
		_, err := create.Execute(ctx, usecase.CreateCalendarInput{ActorID: ana.ID, CalendarFields: usecase.CalendarFields{Name: ptr("X"), Eras: &eras}})
		if domain.KindOf(err) != domain.KindValidation {
			t.Errorf("%s: erro = %v", name, err)
		}
	}
	c := createCalendar(t, d, ana.ID, false)
	for _, e := range c.Calendar.Eras {
		if e.ID == "" {
			t.Fatalf("era sem ID: %+v", e)
		}
	}
	if c.OwnerHandle != "ana" {
		t.Fatalf("dono = %q", c.OwnerHandle)
	}
}

func TestCalendarSharing(t *testing.T) {
	d := newDeps()
	ana, bia := register(t, d, "ana"), register(t, d, "bia")
	private := createCalendar(t, d, ana.ID, false)
	shared := createCalendar(t, d, ana.ID, true)

	list := must[[]usecase.CalendarView](t)(usecase.NewListCalendars(d).Execute(ctx, usecase.ListCalendarsInput{ActorID: bia.ID}))
	if len(list) != 1 || list[0].Calendar.ID != shared.Calendar.ID {
		t.Fatalf("bia vê %v", list)
	}
	if list := must[[]usecase.CalendarView](t)(usecase.NewListCalendars(d).Execute(ctx, usecase.ListCalendarsInput{ActorID: ana.ID})); len(list) != 2 {
		t.Fatalf("ana vê %d", len(list))
	}

	// Calendário privado de outra pessoa não serve para criar timeline.
	_, err := createFictional(t, d, bia.ID, "Gondor", private.Calendar.ID)
	wantKind(t, err, domain.KindValidation)
	tl := must[usecase.TimelineDetailView](t)(createFictional(t, d, bia.ID, "Gondor", shared.Calendar.ID))
	if tl.Calendar == nil || tl.Calendar.ID != shared.Calendar.ID {
		t.Fatalf("calendário da timeline = %+v", tl.Calendar)
	}

	// Só o dono edita; em uso, não dá para apagar.
	_, err = usecase.NewUpdateCalendar(d).Execute(ctx, usecase.UpdateCalendarInput{ActorID: bia.ID, CalendarID: shared.Calendar.ID, CalendarFields: usecase.CalendarFields{Name: ptr("Arda")}})
	wantKind(t, err, domain.KindForbidden)
	_, err = usecase.NewDeleteCalendar(d).Execute(ctx, usecase.CalendarRefInput{ActorID: ana.ID, CalendarID: shared.Calendar.ID})
	wantKind(t, err, domain.KindConflict)
	must[usecase.Empty](t)(usecase.NewDeleteCalendar(d).Execute(ctx, usecase.CalendarRefInput{ActorID: ana.ID, CalendarID: private.Calendar.ID}))
}

func TestFictionalTimelineYears(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	c := createCalendar(t, d, ana.ID, false)

	// Ano 0 não existe no calendário; mês não existe em timeline ficcional.
	_, err := createFictional(t, d, ana.ID, "Anel", c.Calendar.ID, ev(0, "Antes de tudo"))
	wantKind(t, err, domain.KindValidation)
	_, err = createFictional(t, d, ana.ID, "Anel", c.Calendar.ID, domain.EventData{Year: ta3019, Month: ptr(3), Title: "Queda de Sauron"})
	wantKind(t, err, domain.KindValidation)
	// Depois de 3000 vale, porque o eixo é o do calendário.
	must[usecase.TimelineDetailView](t)(createFictional(t, d, ana.ID, "Anel", c.Calendar.ID, ev(ta3019, "Queda de Sauron")))

	// Gregoriana e ficcional não se comparam; duas do mesmo calendário, sim.
	createTimeline(t, d, ana.ID, "Brasil", ev(1500, "Chegada"))
	_, err = usecase.NewCompare(d).Execute(ctx, usecase.CompareInput{ActorID: ana.ID, Slugs: []string{"anel", "brasil"}})
	wantKind(t, err, domain.KindValidation)
	must[usecase.TimelineDetailView](t)(createFictional(t, d, ana.ID, "Gondor", c.Calendar.ID, ev(ta3019, "Coroação de Aragorn")))
	must[usecase.ComparisonView](t)(usecase.NewCompare(d).Execute(ctx, usecase.CompareInput{ActorID: ana.ID, Slugs: []string{"anel", "gondor"}}))
}

func TestCalendarChangeMovesContent(t *testing.T) {
	d := newDeps()
	ana, bia := register(t, d, "ana"), register(t, d, "bia")
	c := createCalendar(t, d, ana.ID, true)
	tl := must[usecase.TimelineDetailView](t)(createFictional(t, d, bia.ID, "Anel", c.Calendar.ID,
		ev(500, "Fim da Primeira Era"),
		domain.EventData{Year: 590 + 1, EndYear: ptr(ta3019), Title: "A história do Anel"},
		ev(ta3019, "Queda de Sauron"),
	))
	eras := []domain.Era{{Name: "Guerra do Anel", Start: ta3019 - 1, End: ptr(ta3019)}}
	must[usecase.TimelineDetailView](t)(usecase.NewUpdateTimeline(d).Execute(ctx, usecase.UpdateTimelineInput{
		ActorID: bia.ID, Slug: tl.Timeline.Slug, TimelineFields: usecase.TimelineFields{Eras: &eras},
	}))
	must[usecase.SuggestionView](t)(usecase.NewCreateSuggestion(d).Execute(ctx, usecase.CreateSuggestionInput{
		ActorID: ana.ID, Slug: "anel", Kind: domain.SuggestionNewEvent, Event: ev(ta3019, "Destruição do Anel"),
	}))

	update := func(eras []domain.CalendarEra) error {
		_, err := usecase.NewUpdateCalendar(d).Execute(ctx, usecase.UpdateCalendarInput{
			ActorID: ana.ID, CalendarID: c.Calendar.ID, CalendarFields: usecase.CalendarFields{Eras: &eras},
		})
		return err
	}
	ids := c.Calendar.Eras

	// A Segunda Era passa a ter 3441 anos e entra uma era nova no começo: o que vem depois anda.
	if err := update([]domain.CalendarEra{
		{Name: "Anos das Árvores", Abbr: "A.Á.", Years: ptr(100)},
		{ID: ids[0].ID, Name: "Primeira Era", Abbr: "P.E.", Years: ptr(590)},
		{ID: ids[1].ID, Name: "Segunda Era", Abbr: "S.E.", Years: ptr(3441)},
		{ID: ids[2].ID, Name: "Terceira Era", Abbr: "T.A.", Years: ptr(3021)},
		{ID: ids[3].ID, Name: "Quarta Era", Abbr: "Q.E."},
	}); err != nil {
		t.Fatal(err)
	}
	page := must[usecase.EventPage](t)(usecase.NewListEvents(d).Execute(ctx, usecase.ListEventsInput{ActorID: bia.ID, Slug: "anel"}))
	got := map[string][2]int{}
	for _, e := range page.Items {
		got[e.Event.Title] = [2]int{e.Event.Year, e.Event.End()}
	}
	newTA := 100 + 590 + 3441 + 3019
	want := map[string][2]int{
		"Fim da Primeira Era": {600, 600},
		"A história do Anel":  {100 + 590 + 1, newTA},
		"Queda de Sauron":     {newTA, newTA},
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, queria %v", k, got[k], v)
		}
	}
	tlAfter := must[usecase.TimelineDetailView](t)(usecase.NewGetTimeline(d).Execute(ctx, usecase.SlugInput{ActorID: bia.ID, Slug: "anel"}))
	if p := tlAfter.Timeline.Eras[0]; p.Start != newTA-1 || *p.End != newTA {
		t.Errorf("período = %d–%d", p.Start, *p.End)
	}
	ss := must[[]usecase.SuggestionView](t)(usecase.NewListSuggestions(d).Execute(ctx, usecase.ListSuggestionsInput{ActorID: bia.ID, Slug: "anel"}))
	if ss[0].Suggestion.Proposed.Year != newTA {
		t.Errorf("sugestão = %d", ss[0].Suggestion.Proposed.Year)
	}

	cur := must[usecase.CalendarView](t)(usecase.NewGetCalendar(d).Execute(ctx, usecase.CalendarRefInput{ActorID: ana.ID, CalendarID: c.Calendar.ID})).Calendar.Eras
	// Curta demais para o que já tem, apagar era com eventos e trocar a ordem: recusados.
	short := append([]domain.CalendarEra{}, cur...)
	short[3].Years = ptr(3000)
	wantKind(t, update(short), domain.KindValidation)
	wantKind(t, update([]domain.CalendarEra{cur[0], cur[2], cur[3], cur[4]}), domain.KindValidation)
	wantKind(t, update([]domain.CalendarEra{cur[0], cur[2], cur[1], cur[3], cur[4]}), domain.KindValidation)
	// Apagar a era vazia dos Anos das Árvores puxa tudo de volta.
	if err := update(cur[1:]); err != nil {
		t.Fatal(err)
	}
	page = must[usecase.EventPage](t)(usecase.NewListEvents(d).Execute(ctx, usecase.ListEventsInput{ActorID: bia.ID, Slug: "anel"}))
	if y := page.Items[len(page.Items)-1].Event.Year; y != 590+3441+3019 {
		t.Errorf("Queda de Sauron = %d", y)
	}
}
