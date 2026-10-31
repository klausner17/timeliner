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

	_, err := setEras([]domain.Era{
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
