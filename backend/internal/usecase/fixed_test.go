package usecase_test

import (
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func createFixed(t *testing.T, d *usecase.Deps, owner, title string, scale domain.Scale, events ...domain.EventData) (usecase.TimelineDetailView, error) {
	t.Helper()
	return usecase.NewCreateTimeline(d).Execute(ctx, usecase.CreateTimelineInput{
		ActorID: owner, Scale: &scale,
		TimelineFields: usecase.TimelineFields{Title: &title, Category: ptr("Natureza"), Cyclic: ptr(true)},
		Events:         events,
	})
}

func TestFixedTimeline(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	lagarto := must[usecase.TimelineDetailView](t)(createFixed(t, d, ana.ID, "Ciclo do lagarto", domain.ScaleDays,
		ev(0, "Ovo"), ev(60, "Eclosão"), ev(365, "Adulto")))
	if lagarto.Timeline.Scale == nil || *lagarto.Timeline.Scale != domain.ScaleDays || !lagarto.Timeline.Cyclic {
		t.Fatalf("timeline = %+v", lagarto.Timeline)
	}
	if s := lagarto.Stats; s.StartYear == nil || *s.StartYear != 0 || *s.EndYear != 365 {
		t.Fatalf("faixa = %v–%v", s.StartYear, s.EndYear)
	}

	// Antes do início, mês e calendário não valem.
	_, err := createFixed(t, d, ana.ID, "X", domain.ScaleDays, ev(-1, "Antes"))
	wantKind(t, err, domain.KindValidation)
	_, err = createFixed(t, d, ana.ID, "X", domain.ScaleDays, domain.EventData{Year: 3, Month: ptr(2), Title: "Mês"})
	wantKind(t, err, domain.KindValidation)
	_, err = createFixed(t, d, ana.ID, "X", "horas")
	wantKind(t, err, domain.KindValidation)
	cal := createCalendar(t, d, ana.ID, false)
	scale := domain.ScaleDays
	_, err = usecase.NewCreateTimeline(d).Execute(ctx, usecase.CreateTimelineInput{
		ActorID: ana.ID, Scale: &scale, CalendarID: cal.Calendar.ID,
		TimelineFields: usecase.TimelineFields{Title: ptr("X"), Category: ptr("Natureza")},
	})
	wantKind(t, err, domain.KindValidation)

	// Períodos (fases) na escala, sem mês.
	eras := []domain.Era{{Name: "Ovo", Start: 0, End: ptr(59)}, {Name: "Filhote", Start: 60, End: ptr(364)}}
	upd := usecase.NewUpdateTimeline(d)
	must[usecase.TimelineDetailView](t)(upd.Execute(ctx, usecase.UpdateTimelineInput{ActorID: ana.ID, Slug: lagarto.Timeline.Slug, TimelineFields: usecase.TimelineFields{Eras: &eras}}))
	bad := []domain.Era{{Name: "Ovo", Start: 0, StartMonth: ptr(1)}}
	_, err = upd.Execute(ctx, usecase.UpdateTimelineInput{ActorID: ana.ID, Slug: lagarto.Timeline.Slug, TimelineFields: usecase.TimelineFields{Eras: &bad}})
	wantKind(t, err, domain.KindValidation)

	// Cíclica só em timeline fixa.
	brasil := createTimeline(t, d, ana.ID, "Brasil", ev(1500, "Cabral"))
	_, err = upd.Execute(ctx, usecase.UpdateTimelineInput{ActorID: ana.ID, Slug: brasil.Timeline.Slug, TimelineFields: usecase.TimelineFields{Cyclic: ptr(true)}})
	wantKind(t, err, domain.KindValidation)
	out := must[usecase.TimelineDetailView](t)(upd.Execute(ctx, usecase.UpdateTimelineInput{ActorID: ana.ID, Slug: lagarto.Timeline.Slug, TimelineFields: usecase.TimelineFields{Cyclic: ptr(false)}}))
	if out.Timeline.Cyclic || out.Timeline.Scale == nil {
		t.Fatalf("depois de editar: %+v", out.Timeline)
	}
}

func TestCompareFixedTimelines(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	must[usecase.TimelineDetailView](t)(createFixed(t, d, ana.ID, "Lagarto", domain.ScaleDays, ev(0, "Ovo"), ev(60, "Eclosão")))
	must[usecase.TimelineDetailView](t)(createFixed(t, d, ana.ID, "Sapo", domain.ScaleDays, ev(0, "Ovo"), ev(10, "Girino")))
	must[usecase.TimelineDetailView](t)(createFixed(t, d, ana.ID, "Elefante", domain.ScaleYears, ev(0, "Nascimento")))
	createTimeline(t, d, ana.ID, "Brasil", ev(10, "Algo"))
	cmp := usecase.NewCompare(d)

	out := must[usecase.ComparisonView](t)(cmp.Execute(ctx, usecase.CompareInput{Slugs: []string{"lagarto", "sapo"}}))
	if out.Start != 0 || out.End != 60 || len(out.Overlaps) != 1 || out.Overlaps[0].Year != 0 {
		t.Fatalf("eixo %d–%d, sobreposições %+v", out.Start, out.End, out.Overlaps)
	}
	_, err := cmp.Execute(ctx, usecase.CompareInput{Slugs: []string{"lagarto", "elefante"}})
	wantKind(t, err, domain.KindValidation)
	_, err = cmp.Execute(ctx, usecase.CompareInput{Slugs: []string{"lagarto", "brasil"}})
	wantKind(t, err, domain.KindValidation)
}
