package usecase_test

import (
	"slices"
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func TestFavoriteTimeline(t *testing.T) {
	d := newDeps()
	ana, beto := register(t, d, "ana"), register(t, d, "beto")
	createTimeline(t, d, ana.ID, "Brasil")
	createTimeline(t, d, ana.ID, "Escândalos")
	privada := createTimeline(t, d, ana.ID, "Privada")
	must[usecase.TimelineDetailView](t)(usecase.NewUpdateTimeline(d).Execute(ctx, usecase.UpdateTimelineInput{
		ActorID: ana.ID, Slug: privada.Timeline.Slug, TimelineFields: usecase.TimelineFields{Visibility: ptr(domain.VisibilityPrivate)},
	}))

	fav := usecase.NewFavoriteTimeline(d)
	for range 2 { // idempotente
		must[usecase.Empty](t)(fav.Execute(ctx, usecase.FavoriteTimelineInput{ActorID: beto.ID, Slug: "escandalos", Favorite: true}))
	}
	// A dona também pode dar estrela nas próprias.
	must[usecase.Empty](t)(fav.Execute(ctx, usecase.FavoriteTimelineInput{ActorID: ana.ID, Slug: "escandalos", Favorite: true}))
	must[usecase.Empty](t)(fav.Execute(ctx, usecase.FavoriteTimelineInput{ActorID: ana.ID, Slug: "brasil", Favorite: true}))

	_, err := fav.Execute(ctx, usecase.FavoriteTimelineInput{Slug: "brasil", Favorite: true})
	wantKind(t, err, domain.KindUnauthorized)
	_, err = fav.Execute(ctx, usecase.FavoriteTimelineInput{ActorID: beto.ID, Slug: "privada", Favorite: true})
	wantKind(t, err, domain.KindNotFound)

	tl := must[usecase.TimelineDetailView](t)(usecase.NewGetTimeline(d).Execute(ctx, usecase.SlugInput{ActorID: beto.ID, Slug: "escandalos"}))
	if tl.Stats.Favorites != 2 || !tl.Stats.Favorited || tl.Stats.Following {
		t.Fatalf("timeline = %+v", tl.Stats)
	}

	list := usecase.NewListTimelines(d)
	slugs := func(in usecase.ListTimelinesInput) []string {
		t.Helper()
		out := []string{}
		for _, v := range must[usecase.TimelinePage](t)(list.Execute(ctx, in)).Items {
			out = append(out, v.Timeline.Slug)
		}
		return out
	}
	// Populares: mais estrelas primeiro.
	if got := slugs(usecase.ListTimelinesInput{}); !slices.Equal(got, []string{"escandalos", "brasil"}) {
		t.Errorf("popular = %v", got)
	}
	// As favoritas de alguém: as favoritadas por último primeiro.
	if got := slugs(usecase.ListTimelinesInput{FavoritedBy: "ana"}); !slices.Equal(got, []string{"brasil", "escandalos"}) {
		t.Errorf("favoritas da ana = %v", got)
	}
	if got := slugs(usecase.ListTimelinesInput{FavoritedBy: "ninguem"}); len(got) != 0 {
		t.Errorf("favoritas de quem não existe = %v", got)
	}

	must[usecase.Empty](t)(fav.Execute(ctx, usecase.FavoriteTimelineInput{ActorID: beto.ID, Slug: "escandalos"}))
	must[usecase.Empty](t)(fav.Execute(ctx, usecase.FavoriteTimelineInput{ActorID: ana.ID, Slug: "escandalos"}))
	if got := slugs(usecase.ListTimelinesInput{}); !slices.Equal(got, []string{"brasil", "escandalos"}) {
		t.Errorf("popular depois de tirar as estrelas = %v", got)
	}
	tl = must[usecase.TimelineDetailView](t)(usecase.NewGetTimeline(d).Execute(ctx, usecase.SlugInput{ActorID: beto.ID, Slug: "escandalos"}))
	if tl.Stats.Favorites != 0 || tl.Stats.Favorited {
		t.Fatalf("sem estrela = %+v", tl.Stats)
	}
}
