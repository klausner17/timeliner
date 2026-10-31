package usecase_test

import (
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func eraOf(evs []usecase.EventView, title string) *string {
	for _, e := range evs {
		if e.Event.Title == title {
			return e.Event.Era
		}
	}
	return nil
}

func TestPeriods(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	createTimeline(t, d, ana.ID, "Brasil")
	update := usecase.NewUpdateTimeline(d)
	setEras := func(eras []domain.Era, renamed map[string]string) (usecase.TimelineDetailView, error) {
		return update.Execute(ctx, usecase.UpdateTimelineInput{
			ActorID: ana.ID, Slug: "brasil", TimelineFields: usecase.TimelineFields{Eras: &eras, RenamedEras: renamed},
		})
	}

	// Sobrepostos e fora de ordem: voltam ordenados pelo início, quem contém antes.
	tl := must[usecase.TimelineDetailView](t)(setEras([]domain.Era{
		{Name: "Estado Novo", Start: 1937, End: ptr(1945), Color: ptr(domain.PeriodColor("vermelho"))},
		{Name: " República Velha ", Start: 1889, End: ptr(1930)},
		{Name: "Era Vargas", Start: 1930, End: ptr(1945)},
		{Name: "Nova República", Start: 1985},
	}, nil))
	var names []string
	for _, e := range tl.Timeline.Eras {
		names = append(names, e.Name)
	}
	if want := []string{"República Velha", "Era Vargas", "Estado Novo", "Nova República"}; len(names) != 4 || names[0] != want[0] || names[1] != want[1] || names[2] != want[2] || names[3] != want[3] {
		t.Fatalf("ordem = %v", names)
	}
	if c := tl.Timeline.Eras[2].Color; c == nil || *c != "vermelho" {
		t.Fatalf("cor = %v", c)
	}

	create := usecase.NewCreateEvent(d)
	for _, e := range []domain.EventData{
		{Year: 1937, Title: "Golpe do Estado Novo", Era: ptr("Estado Novo")},
		{Year: 1930, Title: "Revolução de 1930", Era: ptr("Era Vargas")},
		{Year: 1889, Title: "Proclamação", Era: ptr("República Velha")},
	} {
		must[usecase.EventView](t)(create.Execute(ctx, usecase.CreateEventInput{ActorID: ana.ID, Slug: "brasil", Event: e}))
	}

	// Renomeia um, remove outro: os eventos acompanham.
	must[usecase.TimelineDetailView](t)(setEras([]domain.Era{
		{Name: "Primeira República", Start: 1889, End: ptr(1930)},
		{Name: "Era Vargas", Start: 1930, End: ptr(1945)},
	}, map[string]string{"Primeira República": "República Velha"}))
	evs := must[usecase.EventPage](t)(usecase.NewListEvents(d).Execute(ctx, usecase.ListEventsInput{Slug: "brasil"})).Items
	if e := eraOf(evs, "Proclamação"); e == nil || *e != "Primeira República" {
		t.Fatalf("renomeado: era = %v", e)
	}
	if e := eraOf(evs, "Golpe do Estado Novo"); e != nil {
		t.Fatalf("removido: era = %v", *e)
	}
	if e := eraOf(evs, "Revolução de 1930"); e == nil || *e != "Era Vargas" {
		t.Fatalf("mantido: era = %v", e)
	}

	// Troca de nomes entre dois períodos de uma vez.
	must[usecase.TimelineDetailView](t)(setEras([]domain.Era{
		{Name: "Era Vargas", Start: 1889, End: ptr(1930)},
		{Name: "Primeira República", Start: 1930, End: ptr(1945)},
	}, map[string]string{"Era Vargas": "Primeira República", "Primeira República": "Era Vargas"}))
	evs = must[usecase.EventPage](t)(usecase.NewListEvents(d).Execute(ctx, usecase.ListEventsInput{Slug: "brasil"})).Items
	if e := eraOf(evs, "Proclamação"); e == nil || *e != "Era Vargas" {
		t.Fatalf("troca: era = %v", e)
	}

	// Com dia e mês: no mesmo ano, quem começa antes vem antes.
	tl = must[usecase.TimelineDetailView](t)(setEras([]domain.Era{
		{Name: "Era Vargas", Start: 1930, StartMonth: ptr(11), StartDay: ptr(3), End: ptr(1945), EndMonth: ptr(10), EndDay: ptr(29)},
		{Name: "Primeira República", Start: 1889, StartMonth: ptr(11), StartDay: ptr(15), End: ptr(1930), EndMonth: ptr(10), EndDay: ptr(24)},
		{Name: "Revolução", Start: 1930, StartMonth: ptr(10), End: ptr(1930), EndMonth: ptr(11)},
	}, nil))
	if e := tl.Timeline.Eras; e[1].Name != "Revolução" || *e[2].StartDay != 3 || *e[0].EndMonth != 10 {
		t.Fatalf("eras com data = %+v", e)
	}

	_, err := setEras([]domain.Era{
		{Name: "D1", Start: 1930, StartMonth: ptr(2), StartDay: ptr(30)},
		{Name: "D2", Start: 1930, StartDay: ptr(3)},
		{Name: "D3", Start: 1930, StartMonth: ptr(5), End: ptr(1930), EndMonth: ptr(4)},
		{Name: "D4", Start: 1930, EndMonth: ptr(4)},
		{Name: "D5", Start: 1930, StartMonth: ptr(5), End: ptr(1930)},
	}, nil)
	wantKind(t, err, domain.KindValidation)
	fields := err.(*domain.Error).Fields
	for _, f := range []string{"eras[0].startDay", "eras[1].startDay", "eras[2].end", "eras[3].endMonth"} {
		if _, ok := fields[f]; !ok {
			t.Errorf("faltou erro em %s: %v", f, fields)
		}
	}
	if len(fields) != 4 {
		t.Errorf("fim só com o ano cobre o ano inteiro; erros = %v", fields)
	}

	_, err = setEras([]domain.Era{
		{Name: "A", Start: 1900, End: ptr(1800)},
		{Name: "A", Start: 1900},
		{Name: "B", Start: 1900, Color: ptr(domain.PeriodColor("rosa-choque"))},
	}, nil)
	wantKind(t, err, domain.KindValidation)
	for _, f := range []string{"eras[0].end", "eras[1].name", "eras[2].color"} {
		if _, ok := err.(*domain.Error).Fields[f]; !ok {
			t.Errorf("faltou erro em %s: %v", f, err.(*domain.Error).Fields)
		}
	}
}
