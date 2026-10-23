package usecase_test

import (
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
		ActorID: ana.ID, Slug: "brasil", Event: domain.EventData{Year: 1964, Title: " Golpe ", Era: ptr("Ditadura"), Date: ptr("")},
	}))
	if golpe.Event.Title != "Golpe" || golpe.Event.Date != nil || golpe.AuthorHandle != "ana" {
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
