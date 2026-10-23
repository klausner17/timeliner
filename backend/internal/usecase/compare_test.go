package usecase_test

import (
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func TestCompare(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	createTimeline(t, d, ana.ID, "Brasil", ev(1985, "Redemocratização"), ev(1992, "Impeachment"))
	createTimeline(t, d, ana.ID, "Escândalos", ev(1992, "PC Farias"), ev(2005, "Mensalão"))
	cmp := usecase.NewCompare(d)

	out := must[usecase.ComparisonView](t)(cmp.Execute(ctx, usecase.CompareInput{Slugs: []string{"brasil", "escandalos"}}))
	if out.Start != 1985 || out.End != 2005 || len(out.Lanes) != 2 {
		t.Fatalf("eixo = %d–%d, %d trilhas", out.Start, out.End, len(out.Lanes))
	}
	if out.Lanes[0].Color != domain.ColorA || out.Lanes[1].Color != domain.ColorB {
		t.Fatalf("cores = %s %s", out.Lanes[0].Color, out.Lanes[1].Color)
	}
	if len(out.Overlaps) != 1 || out.Overlaps[0].Year != 1992 || len(out.Overlaps[0].Slugs) != 2 {
		t.Fatalf("sobreposições = %+v", out.Overlaps)
	}

	out = must[usecase.ComparisonView](t)(cmp.Execute(ctx, usecase.CompareInput{Slugs: []string{"brasil", "escandalos"}, Start: ptr(1990), End: ptr(2000)}))
	if len(out.Lanes[0].Events) != 1 || len(out.Lanes[1].Events) != 1 {
		t.Fatalf("filtro de anos: %d e %d eventos", len(out.Lanes[0].Events), len(out.Lanes[1].Events))
	}

	_, err := cmp.Execute(ctx, usecase.CompareInput{Slugs: []string{"brasil", "brasil"}})
	wantKind(t, err, domain.KindValidation)
	_, err = cmp.Execute(ctx, usecase.CompareInput{Slugs: []string{"brasil", "nada"}})
	wantKind(t, err, domain.KindNotFound)
}

func TestSavedComparisons(t *testing.T) {
	d := newDeps()
	ana, beto := register(t, d, "ana"), register(t, d, "beto")
	createTimeline(t, d, ana.ID, "Brasil")
	createTimeline(t, d, ana.ID, "Escândalos")

	save := usecase.NewSaveComparison(d)
	c := must[domain.SavedComparison](t)(save.Execute(ctx, usecase.SaveComparisonInput{ActorID: ana.ID, Slugs: []string{"brasil", "escandalos"}, Start: 1980, End: 2020}))
	_, err := save.Execute(ctx, usecase.SaveComparisonInput{ActorID: ana.ID, Slugs: []string{"brasil", "sumiu"}, Start: 1980, End: 2020})
	wantKind(t, err, domain.KindValidation)

	list := must[[]domain.SavedComparison](t)(usecase.NewListSavedComparisons(d).Execute(ctx, usecase.ActorInput{ActorID: ana.ID}))
	if len(list) != 1 || list[0].ID != c.ID {
		t.Fatalf("salvas = %+v", list)
	}
	del := usecase.NewDeleteSavedComparison(d)
	_, err = del.Execute(ctx, usecase.DeleteSavedComparisonInput{ActorID: beto.ID, ComparisonID: c.ID})
	wantKind(t, err, domain.KindNotFound)
	must[usecase.Empty](t)(del.Execute(ctx, usecase.DeleteSavedComparisonInput{ActorID: ana.ID, ComparisonID: c.ID}))
}
