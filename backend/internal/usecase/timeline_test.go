package usecase_test

import (
	"slices"
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func TestCreateTimelineSlugAndStats(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	tl := createTimeline(t, d, ana.ID, "História do Brasil", ev(1500, "Chegada"), ev(1822, "Independência"), domain.EventData{Year: 1964, EndYear: ptr(1985), Title: "Ditadura"})
	if tl.Timeline.Slug != "historia-do-brasil" {
		t.Fatalf("slug = %q", tl.Timeline.Slug)
	}
	if tl.Stats.EventCount != 3 || *tl.Stats.StartYear != 1500 || *tl.Stats.EndYear != 1985 {
		t.Fatalf("stats = %+v", tl.Stats)
	}
	if !slices.Equal(tl.Points, []float64{0, 66.4, 95.7}) {
		t.Fatalf("points = %v", tl.Points)
	}
	if len(tl.Events) != 3 || tl.Events[0].AuthorHandle != "ana" {
		t.Fatalf("eventos = %+v", tl.Events)
	}
	again := createTimeline(t, d, ana.ID, "História do Brasil")
	if again.Timeline.Slug != "historia-do-brasil-2" {
		t.Fatalf("slug repetido = %q", again.Timeline.Slug)
	}
}

func TestCreateTimelineValidation(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	_, err := usecase.NewCreateTimeline(d).Execute(ctx, usecase.CreateTimelineInput{
		ActorID:        ana.ID,
		TimelineFields: usecase.TimelineFields{Title: ptr(" "), Category: ptr("Futebol"), Color: ptr(domain.TrackColor("z"))},
		Events:         []domain.EventData{{Year: 2000, Title: "ok"}, {Year: 2000, EndYear: ptr(1990), Title: "", Era: ptr("Nada")}},
	})
	wantKind(t, err, domain.KindValidation)
	fields := err.(*domain.Error).Fields
	for _, f := range []string{"title", "category", "color", "events[1].title", "events[1].endYear", "events[1].era"} {
		if fields[f] == "" {
			t.Errorf("faltou erro em %s: %v", f, fields)
		}
	}
	_, err = usecase.NewCreateTimeline(d).Execute(ctx, usecase.CreateTimelineInput{})
	wantKind(t, err, domain.KindUnauthorized)
}

func TestTimelineVisibilityAndOwnership(t *testing.T) {
	d := newDeps()
	ana, beto := register(t, d, "ana"), register(t, d, "beto")
	tl := createTimeline(t, d, ana.ID, "Diário")
	slug := tl.Timeline.Slug

	_, err := usecase.NewUpdateTimeline(d).Execute(ctx, usecase.UpdateTimelineInput{ActorID: beto.ID, Slug: slug, TimelineFields: usecase.TimelineFields{Title: ptr("x")}})
	wantKind(t, err, domain.KindForbidden)

	must[usecase.TimelineDetailView](t)(usecase.NewUpdateTimeline(d).Execute(ctx, usecase.UpdateTimelineInput{
		ActorID: ana.ID, Slug: slug, TimelineFields: usecase.TimelineFields{Visibility: ptr(domain.VisibilityPrivate), Tags: &[]string{"Vida", "vida ", ""}},
	}))
	_, err = usecase.NewGetTimeline(d).Execute(ctx, usecase.SlugInput{ActorID: beto.ID, Slug: slug})
	wantKind(t, err, domain.KindNotFound)
	got := must[usecase.TimelineDetailView](t)(usecase.NewGetTimeline(d).Execute(ctx, usecase.SlugInput{ActorID: ana.ID, Slug: slug}))
	if !slices.Equal(got.Timeline.Tags, []string{"vida"}) {
		t.Fatalf("tags = %v", got.Timeline.Tags)
	}

	_, err = usecase.NewDeleteTimeline(d).Execute(ctx, usecase.SlugInput{ActorID: beto.ID, Slug: slug})
	wantKind(t, err, domain.KindNotFound)
	must[usecase.Empty](t)(usecase.NewDeleteTimeline(d).Execute(ctx, usecase.SlugInput{ActorID: ana.ID, Slug: slug}))
	_, err = usecase.NewGetTimeline(d).Execute(ctx, usecase.SlugInput{ActorID: ana.ID, Slug: slug})
	wantKind(t, err, domain.KindNotFound)
}

func TestListTimelines(t *testing.T) {
	d := newDeps()
	ana, beto := register(t, d, "ana"), register(t, d, "beto")
	createTimeline(t, d, ana.ID, "Brasil")
	escandalos := createTimeline(t, d, ana.ID, "Escândalos")
	rascunho := createTimeline(t, d, ana.ID, "Rascunho")
	must[usecase.TimelineDetailView](t)(usecase.NewUpdateTimeline(d).Execute(ctx, usecase.UpdateTimelineInput{
		ActorID: ana.ID, Slug: rascunho.Timeline.Slug, TimelineFields: usecase.TimelineFields{Draft: ptr(true)},
	}))
	must[usecase.TimelineDetailView](t)(usecase.NewUpdateTimeline(d).Execute(ctx, usecase.UpdateTimelineInput{
		ActorID: ana.ID, Slug: escandalos.Timeline.Slug, TimelineFields: usecase.TimelineFields{Category: ptr("Política"), Contributions: ptr(domain.ContributionsClosed)},
	}))
	must[usecase.Empty](t)(usecase.NewFollowTimeline(d).Execute(ctx, usecase.FollowTimelineInput{ActorID: beto.ID, Slug: escandalos.Timeline.Slug, Follow: true}))

	list := usecase.NewListTimelines(d)
	slugs := func(in usecase.ListTimelinesInput) []string {
		t.Helper()
		page := must[usecase.TimelinePage](t)(list.Execute(ctx, in))
		out := []string{}
		for _, v := range page.Items {
			out = append(out, v.Timeline.Slug)
		}
		return out
	}
	if got := slugs(usecase.ListTimelinesInput{}); !slices.Equal(got, []string{"escandalos", "brasil"}) {
		t.Errorf("popular = %v", got)
	}
	if got := slugs(usecase.ListTimelinesInput{Category: "Política"}); !slices.Equal(got, []string{"escandalos"}) {
		t.Errorf("cat = %v", got)
	}
	if got := slugs(usecase.ListTimelinesInput{OnlyOpen: true}); !slices.Equal(got, []string{"brasil"}) {
		t.Errorf("open = %v", got)
	}
	if got := slugs(usecase.ListTimelinesInput{ActorID: ana.ID, OwnerHandle: "me", Sort: usecase.SortRecent}); !slices.Equal(got, []string{"rascunho", "escandalos", "brasil"}) {
		t.Errorf("minhas = %v", got)
	}
	if got := slugs(usecase.ListTimelinesInput{ActorID: beto.ID, OwnerHandle: "ana", Sort: usecase.SortRecent}); !slices.Equal(got, []string{"escandalos", "brasil"}) {
		t.Errorf("da ana vistas pelo beto = %v", got)
	}
	page := must[usecase.TimelinePage](t)(list.Execute(ctx, usecase.ListTimelinesInput{Page: 2, PageSize: 1, Sort: usecase.SortRecent}))
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].Timeline.Slug != "brasil" {
		t.Errorf("página 2 = %+v", page)
	}
	_, err := list.Execute(ctx, usecase.ListTimelinesInput{PageSize: 500})
	wantKind(t, err, domain.KindValidation)
}
