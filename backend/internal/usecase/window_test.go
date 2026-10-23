package usecase_test

import (
	"fmt"
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

// manyEvents cria n eventos espalhados de from em diante, step anos entre eles.
func manyEvents(n, from, step int) []domain.EventData {
	out := make([]domain.EventData, n)
	for i := range out {
		out[i] = ev(from+i*step, fmt.Sprintf("Evento %d", i))
	}
	return out
}

func TestListEventsPaginates(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	// Vários no mesmo ano: o cursor precisa desempatar sem repetir nem pular.
	events := append(manyEvents(7, 1900, 0), manyEvents(5, 1950, 10)...)
	tl := createTimeline(t, d, ana.ID, "Brasil", events...)
	if len(tl.Events.Items) != len(events) || tl.Events.NextCursor != nil {
		t.Fatalf("detalhe com %d eventos e cursor %v", len(tl.Events.Items), tl.Events.NextCursor)
	}

	list := usecase.NewListEvents(d)
	seen := map[string]bool{}
	pages := 0
	cursor := ""
	for {
		page := must[usecase.EventPage](t)(list.Execute(ctx, usecase.ListEventsInput{Slug: "brasil", Limit: 5, Cursor: cursor}))
		pages++
		for _, e := range page.Items {
			if seen[e.Event.ID] {
				t.Fatalf("evento %s repetido na página %d", e.Event.ID, pages)
			}
			seen[e.Event.ID] = true
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	if pages != 3 || len(seen) != len(events) {
		t.Fatalf("%d páginas, %d eventos", pages, len(seen))
	}

	from := 1960
	page := must[usecase.EventPage](t)(list.Execute(ctx, usecase.ListEventsInput{Slug: "brasil", From: &from, Limit: 2}))
	if len(page.Items) != 2 || page.Items[0].Event.Year != 1960 || page.NextCursor == nil {
		t.Fatalf("from=1960: %+v", page)
	}

	_, err := list.Execute(ctx, usecase.ListEventsInput{Slug: "brasil", Cursor: "nao-e-cursor"})
	wantKind(t, err, domain.KindValidation)
	_, err = list.Execute(ctx, usecase.ListEventsInput{Slug: "brasil", Limit: usecase.MaxPageSize + 1})
	wantKind(t, err, domain.KindValidation)
}

func TestTimelineDetailHasFirstPage(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	tl := createTimeline(t, d, ana.ID, "Grande", manyEvents(usecase.DefaultPageSize+20, 1000, 1)...)
	if len(tl.Events.Items) != usecase.DefaultPageSize || tl.Events.NextCursor == nil || tl.Stats.EventCount != usecase.DefaultPageSize+20 {
		t.Fatalf("detalhe com %d eventos, cursor %v, total %d", len(tl.Events.Items), tl.Events.NextCursor, tl.Stats.EventCount)
	}
	if len(tl.Points) != 24 || tl.Points[0] != 0 || tl.Points[23] != 100 {
		t.Fatalf("points = %v", tl.Points)
	}
	rest := must[usecase.EventPage](t)(usecase.NewListEvents(d).Execute(ctx, usecase.ListEventsInput{Slug: "grande", Cursor: *tl.Events.NextCursor}))
	if len(rest.Items) != 20 || rest.Items[0].Event.Year != 1000+usecase.DefaultPageSize || rest.NextCursor != nil {
		t.Fatalf("segunda página: %d eventos, primeiro em %d", len(rest.Items), rest.Items[0].Event.Year)
	}
}

func TestCreateTimelineImportLimit(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	_, err := usecase.NewCreateTimeline(d).Execute(ctx, usecase.CreateTimelineInput{
		ActorID: ana.ID, TimelineFields: usecase.TimelineFields{Title: ptr("Demais"), Category: ptr("História")},
		Events: manyEvents(usecase.MaxImport+1, 1000, 1),
	})
	wantKind(t, err, domain.KindValidation)
	if e := err.(*domain.Error); e.Fields["events"] == "" {
		t.Fatalf("campos = %v", e.Fields)
	}
}

func TestSummarizeEvents(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	// Poucos eventos num trecho enorme: vêm todos, sem agrupar.
	createTimeline(t, d, ana.ID, "Esparsa", ev(-500, "Antes"), ev(1500, "Depois"))
	sum := usecase.NewSummarizeEvents(d)
	w := must[usecase.EventWindow](t)(sum.Execute(ctx, usecase.SummarizeEventsInput{Slug: "esparsa"}))
	if w.Mode != usecase.WindowEvents || w.Total != 2 || len(w.Events) != 2 || w.From != -500 || w.To != 1500 {
		t.Fatalf("esparsa = %+v", w)
	}

	// 300 eventos, um por ano de 1700 a 1999, em 10 faixas de 30 anos.
	createTimeline(t, d, ana.ID, "Densa", manyEvents(300, 1700, 1)...)
	w = must[usecase.EventWindow](t)(sum.Execute(ctx, usecase.SummarizeEventsInput{Slug: "densa", WindowOptions: usecase.WindowOptions{Buckets: 10}}))
	if w.Mode != usecase.WindowBuckets || w.Total != 300 || len(w.Buckets) != 10 || len(w.Events) != 0 {
		t.Fatalf("densa: modo %s, total %d, %d faixas", w.Mode, w.Total, len(w.Buckets))
	}
	b := w.Buckets[1]
	if b.From != 1730 || b.To != 1759 || b.Count != 30 || len(b.Sample) != 3 || b.Sample[0].Event.Year != 1730 || b.Sample[0].AuthorHandle != "ana" {
		t.Fatalf("faixa 1 = %d–%d, %d eventos, exemplos %+v", b.From, b.To, b.Count, b.Sample)
	}

	// Com zoom num trecho pequeno, os eventos voltam a vir um a um.
	w = must[usecase.EventWindow](t)(sum.Execute(ctx, usecase.SummarizeEventsInput{Slug: "densa", From: ptr(1800), To: ptr(1819)}))
	if w.Mode != usecase.WindowEvents || w.Total != 20 {
		t.Fatalf("zoom = %s com %d", w.Mode, w.Total)
	}
	// maxEvents decide o corte.
	w = must[usecase.EventWindow](t)(sum.Execute(ctx, usecase.SummarizeEventsInput{Slug: "densa", From: ptr(1800), To: ptr(1819), WindowOptions: usecase.WindowOptions{MaxEvents: 10}}))
	if w.Mode != usecase.WindowBuckets || w.Total != 20 {
		t.Fatalf("maxEvents=10 = %s com %d", w.Mode, w.Total)
	}
	// Faixas não ficam menores que um ano.
	w = must[usecase.EventWindow](t)(sum.Execute(ctx, usecase.SummarizeEventsInput{Slug: "densa", From: ptr(1800), To: ptr(1804), WindowOptions: usecase.WindowOptions{MaxEvents: 1, Buckets: 50}}))
	if len(w.Buckets) != 5 || w.Buckets[4].From != 1804 || w.Buckets[4].To != 1804 {
		t.Fatalf("faixas de um ano = %+v", w.Buckets)
	}

	_, err := sum.Execute(ctx, usecase.SummarizeEventsInput{Slug: "densa", WindowOptions: usecase.WindowOptions{Buckets: usecase.MaxBuckets + 1}})
	wantKind(t, err, domain.KindValidation)
	_, err = sum.Execute(ctx, usecase.SummarizeEventsInput{Slug: "densa", From: ptr(1900), To: ptr(1800)})
	wantKind(t, err, domain.KindValidation)
}

func TestCompareLevelOfDetail(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	createTimeline(t, d, ana.ID, "Densa", manyEvents(400, 1600, 1)...)
	createTimeline(t, d, ana.ID, "Pouca", ev(1700, "Um"), ev(1800, "Dois"), ev(2100, "Fora"))
	out := must[usecase.ComparisonView](t)(usecase.NewCompare(d).Execute(ctx, usecase.CompareInput{
		Slugs: []string{"pouca", "densa"}, Start: ptr(1600), End: ptr(1999), WindowOptions: usecase.WindowOptions{Buckets: 20},
	}))
	pouca, densa := out.Lanes[0].Window, out.Lanes[1].Window
	if pouca.Mode != usecase.WindowEvents || pouca.Total != 2 {
		t.Fatalf("pouca = %s com %d", pouca.Mode, pouca.Total)
	}
	if densa.Mode != usecase.WindowBuckets || densa.Total != 400 || len(densa.Buckets) != 20 {
		t.Fatalf("densa = %s com %d em %d faixas", densa.Mode, densa.Total, len(densa.Buckets))
	}
	if out.OverlapsTotal != 2 || len(out.Overlaps) != 2 || out.Overlaps[0].Year != 1700 || out.Overlaps[0].Slugs[0] != "pouca" {
		t.Fatalf("sobreposições = %d %+v", out.OverlapsTotal, out.Overlaps)
	}
}
