package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

type fakePreviews map[string]domain.LinkPreview

func (f fakePreviews) Preview(_ context.Context, u string) (domain.LinkPreview, error) {
	p, ok := f[u]
	if !ok {
		return domain.LinkPreview{}, errors.New("fora do ar")
	}
	return p, nil
}

func TestEventLinks(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	createTimeline(t, d, ana.ID, "Brasil")
	create := usecase.NewCreateEvent(d)

	_, err := create.Execute(ctx, usecase.CreateEventInput{ActorID: ana.ID, Slug: "brasil", Event: domain.EventData{
		Year: 1888, Title: "Lei Áurea", Links: []domain.EventLink{{URL: "javascript:alert(1)"}},
	}})
	wantKind(t, err, domain.KindValidation)
	if f := err.(*domain.Error).Fields; f["links[0].url"] == "" {
		t.Fatalf("campos = %v", f)
	}

	lei := must[usecase.EventView](t)(create.Execute(ctx, usecase.CreateEventInput{ActorID: ana.ID, Slug: "brasil", Event: domain.EventData{
		Year: 1888, Title: "Lei Áurea", SourceURL: ptr("https://pt.wikipedia.org/wiki/Lei_Áurea"),
		Links: []domain.EventLink{
			{URL: " https://exemplo.com/arquivo ", Label: " Arquivo Nacional "},
			{URL: "  "},
			{URL: "https://pt.wikipedia.org/wiki/Lei_Áurea"},
			{URL: "https://fora.do.ar/x"},
		},
	}}))
	if len(lei.Event.Links) != 3 || lei.Event.Links[0] != (domain.EventLink{URL: "https://exemplo.com/arquivo", Label: "Arquivo Nacional"}) {
		t.Fatalf("links = %+v", lei.Event.Links)
	}

	// Sem buscador, só os endereços; fonte primeiro e sem repetir.
	preview := usecase.NewPreviewEventLinks(d)
	ps := must[[]domain.LinkPreview](t)(preview.Execute(ctx, usecase.EventRefInput{Slug: "brasil", EventID: lei.Event.ID}))
	want := []string{"https://pt.wikipedia.org/wiki/Lei_Áurea", "https://exemplo.com/arquivo", "https://fora.do.ar/x"}
	if len(ps) != len(want) {
		t.Fatalf("prévias = %+v", ps)
	}
	for i, u := range want {
		if ps[i].URL != u || ps[i].Title != "" {
			t.Fatalf("prévia %d = %+v, queria só %s", i, ps[i], u)
		}
	}

	d.Previews = fakePreviews{
		"https://exemplo.com/arquivo": {Title: "Arquivo", Image: "https://exemplo.com/foto.jpg"},
	}
	ps = must[[]domain.LinkPreview](t)(preview.Execute(ctx, usecase.EventRefInput{Slug: "brasil", EventID: lei.Event.ID}))
	if ps[1].Title != "Arquivo" || ps[1].URL != "https://exemplo.com/arquivo" || ps[2].Title != "" || ps[2].URL != "https://fora.do.ar/x" {
		t.Fatalf("prévias = %+v", ps)
	}

	// PATCH com lista vazia apaga os links.
	up := must[usecase.EventView](t)(usecase.NewUpdateEvent(d).Execute(ctx, usecase.UpdateEventInput{
		ActorID: ana.ID, Slug: "brasil", EventID: lei.Event.ID, Patch: usecase.EventPatch{Links: &[]domain.EventLink{}},
	}))
	if len(up.Event.Links) != 0 {
		t.Fatalf("links = %+v", up.Event.Links)
	}

	// Na criação da timeline os links vão junto com os eventos.
	outra := createTimeline(t, d, ana.ID, "Outra", domain.EventData{Year: 1, Title: "E", Links: []domain.EventLink{{URL: "https://a.b/c", Label: "C"}}})
	evs := must[usecase.EventPage](t)(usecase.NewListEvents(d).Execute(ctx, usecase.ListEventsInput{Slug: outra.Timeline.Slug})).Items
	if len(evs) != 1 || len(evs[0].Event.Links) != 1 || evs[0].Event.Links[0].Label != "C" {
		t.Fatalf("eventos = %+v", evs)
	}

	_, err = preview.Execute(ctx, usecase.EventRefInput{Slug: "brasil", EventID: "ev_nao_existe"})
	wantKind(t, err, domain.KindNotFound)
}
