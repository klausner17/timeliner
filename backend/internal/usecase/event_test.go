package usecase_test

import (
	"strings"
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func TestEventsCRUD(t *testing.T) {
	d := newDeps()
	ana, beto := register(t, d, "ana"), register(t, d, "beto")
	createTimeline(t, d, ana.ID, "Brasil", ev(1822, "Independência"))
	must[usecase.TimelineDetailView](t)(usecase.NewUpdateTimeline(d).Execute(ctx, usecase.UpdateTimelineInput{
		ActorID: ana.ID, Slug: "brasil", TimelineFields: usecase.TimelineFields{Eras: &[]domain.Era{{Name: "Ditadura", Start: 1964, End: ptr(1985)}}},
	}))

	create := usecase.NewCreateEvent(d)
	_, err := create.Execute(ctx, usecase.CreateEventInput{ActorID: beto.ID, Slug: "brasil", Event: ev(1964, "Golpe")})
	wantKind(t, err, domain.KindForbidden)
	golpe := must[usecase.EventView](t)(create.Execute(ctx, usecase.CreateEventInput{
		ActorID: ana.ID, Slug: "brasil", Event: domain.EventData{Year: 1964, Title: " Golpe ", Era: ptr("Ditadura")},
	}))
	if golpe.Event.Title != "Golpe" || golpe.AuthorHandle != "ana" {
		t.Fatalf("evento = %+v", golpe)
	}

	updated := must[usecase.EventView](t)(usecase.NewUpdateEvent(d).Execute(ctx, usecase.UpdateEventInput{
		ActorID: ana.ID, Slug: "brasil", EventID: golpe.Event.ID,
		Patch: usecase.EventPatch{EndYear: ptr(ptr(1985)), Era: ptr[*string](nil)},
	}))
	if *updated.Event.EndYear != 1985 || updated.Event.Era != nil || updated.Event.Title != "Golpe" {
		t.Fatalf("atualizado = %+v", updated.Event)
	}

	from := 1970
	evs := must[usecase.EventPage](t)(usecase.NewListEvents(d).Execute(ctx, usecase.ListEventsInput{Slug: "brasil", From: &from})).Items
	if len(evs) != 1 || evs[0].Event.ID != golpe.Event.ID {
		t.Fatalf("período 1964–1985 devia entrar em from=1970: %+v", evs)
	}

	must[usecase.Empty](t)(usecase.NewDeleteEvent(d).Execute(ctx, usecase.EventRefInput{ActorID: ana.ID, Slug: "brasil", EventID: golpe.Event.ID}))
	_, err = usecase.NewDeleteEvent(d).Execute(ctx, usecase.EventRefInput{ActorID: ana.ID, Slug: "brasil", EventID: golpe.Event.ID})
	wantKind(t, err, domain.KindNotFound)
}

func TestUpdateProfile(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	up := usecase.NewUpdateProfile(d)
	u := must[domain.User](t)(up.Execute(ctx, usecase.UpdateProfileInput{ActorID: ana.ID, Bio: ptr(" Historiadora "), Tint: ptr(domain.Tint("b"))}))
	if u.Bio != "Historiadora" || u.Tint != "b" {
		t.Fatalf("perfil = %+v", u)
	}
	_, err := up.Execute(ctx, usecase.UpdateProfileInput{ActorID: ana.ID, Tint: ptr(domain.Tint("rosa"))})
	wantKind(t, err, domain.KindValidation)
	me := must[domain.User](t)(usecase.NewGetMe(d).Execute(ctx, usecase.ActorInput{ActorID: ana.ID}))
	if me.Tint != "b" {
		t.Fatalf("me = %+v", me)
	}
}

func TestEventsFollowMonthDayAndTime(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	on := func(title string, month, day int, at string) domain.EventData {
		e := domain.EventData{Year: 1961, Title: title, Month: &month, Day: &day}
		if at != "" {
			e.Time = &at
		}
		return e
	}
	goulart := on("Goulart", 9, 7, "")
	goulart.EndYear = ptr(1964)
	createTimeline(t, d, ana.ID, "Presidentes",
		goulart, on("Mazzilli", 8, 25, ""), on("Renúncia", 8, 25, "15:00"), on("Jânio", 1, 31, "09:30"),
		domain.EventData{Year: 1961, Title: "Ano de 1961"}, domain.EventData{Year: 1961, Month: ptr(1), Title: "Janeiro"})

	got := []string{}
	cursor := ""
	for {
		page := must[usecase.EventPage](t)(usecase.NewListEvents(d).Execute(ctx, usecase.ListEventsInput{Slug: "presidentes", Cursor: cursor, Limit: 2}))
		for _, e := range page.Items {
			got = append(got, e.Event.Title)
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	want := []string{"Ano de 1961", "Janeiro", "Jânio", "Mazzilli", "Renúncia", "Goulart"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("ordem = %v, queria %v", got, want)
	}
}

func TestEventMomentValidation(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	createTimeline(t, d, ana.ID, "Brasil")
	create := usecase.NewCreateEvent(d)
	cases := map[string]struct {
		event domain.EventData
		field string
	}{
		"dia sem mês":        {domain.EventData{Year: 1900, Day: ptr(3)}, "day"},
		"29 de fevereiro":    {domain.EventData{Year: 1900, Month: ptr(2), Day: ptr(29)}, "day"},
		"29/02 de 4 a.C.":    {domain.EventData{Year: -4, Month: ptr(2), Day: ptr(29)}, "day"},
		"mês 13":             {domain.EventData{Year: 1900, Month: ptr(13)}, "month"},
		"hora sem dia":       {domain.EventData{Year: 1900, Month: ptr(2), Time: ptr("10:00")}, "time"},
		"hora fora do range": {domain.EventData{Year: 1900, Month: ptr(2), Day: ptr(3), Time: ptr("24:00")}, "time"},
	}
	for name, c := range cases {
		c.event.Title = name
		_, err := create.Execute(ctx, usecase.CreateEventInput{ActorID: ana.ID, Slug: "brasil", Event: c.event})
		wantKind(t, err, domain.KindValidation)
		if err.(*domain.Error).Fields[c.field] == "" {
			t.Fatalf("%s: faltou erro em %q: %v", name, c.field, err.(*domain.Error).Fields)
		}
	}

	leap := must[usecase.EventView](t)(create.Execute(ctx, usecase.CreateEventInput{ActorID: ana.ID, Slug: "brasil",
		Event: domain.EventData{Year: 2024, Month: ptr(2), Day: ptr(29), Time: ptr(" "), Title: "Bissexto"}}))
	if leap.Event.Time != nil {
		t.Fatalf("hora em branco devia virar nil: %+v", leap.Event)
	}
	for _, year := range []int{-1, -5} {
		must[usecase.EventView](t)(create.Execute(ctx, usecase.CreateEventInput{ActorID: ana.ID, Slug: "brasil",
			Event: domain.EventData{Year: year, Month: ptr(2), Day: ptr(29), Title: "Bissexto a.C."}}))
	}
	updated := must[usecase.EventView](t)(usecase.NewUpdateEvent(d).Execute(ctx, usecase.UpdateEventInput{
		ActorID: ana.ID, Slug: "brasil", EventID: leap.Event.ID, Patch: usecase.EventPatch{Time: ptr(ptr("23:59"))},
	}))
	if *updated.Event.Time != "23:59" || *updated.Event.Day != 29 {
		t.Fatalf("atualizado = %+v", updated.Event)
	}
}
