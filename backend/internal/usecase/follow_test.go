package usecase_test

import (
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func TestFollow(t *testing.T) {
	d := newDeps()
	ana, beto := register(t, d, "ana"), register(t, d, "beto")
	createTimeline(t, d, ana.ID, "Brasil")

	ft, fu := usecase.NewFollowTimeline(d), usecase.NewFollowUser(d)
	for range 2 { // idempotente
		must[usecase.Empty](t)(ft.Execute(ctx, usecase.FollowTimelineInput{ActorID: beto.ID, Slug: "brasil", Follow: true}))
		must[usecase.Empty](t)(fu.Execute(ctx, usecase.FollowUserInput{ActorID: beto.ID, Handle: "ana", Follow: true}))
	}
	_, err := fu.Execute(ctx, usecase.FollowUserInput{ActorID: ana.ID, Handle: "ana", Follow: true})
	wantKind(t, err, domain.KindValidation)

	profile := must[usecase.UserView](t)(usecase.NewGetProfile(d).Execute(ctx, usecase.HandleInput{ActorID: beto.ID, Handle: "ana"}))
	if profile.Stats.Followers != 1 || !profile.Stats.Following || profile.Stats.Timelines != 1 {
		t.Fatalf("perfil = %+v", profile.Stats)
	}
	tl := must[usecase.TimelineDetailView](t)(usecase.NewGetTimeline(d).Execute(ctx, usecase.SlugInput{ActorID: beto.ID, Slug: "brasil"}))
	if tl.Stats.Followers != 1 || !tl.Stats.Following {
		t.Fatalf("timeline = %+v", tl.Stats)
	}
	following := must[usecase.FollowingView](t)(usecase.NewListFollowing(d).Execute(ctx, usecase.HandleInput{Handle: "beto"}))
	if len(following.Users) != 1 || len(following.Timelines) != 1 {
		t.Fatalf("seguindo = %+v", following)
	}

	must[usecase.Empty](t)(ft.Execute(ctx, usecase.FollowTimelineInput{ActorID: beto.ID, Slug: "brasil"}))
	tl = must[usecase.TimelineDetailView](t)(usecase.NewGetTimeline(d).Execute(ctx, usecase.SlugInput{ActorID: beto.ID, Slug: "brasil"}))
	if tl.Stats.Followers != 0 || tl.Stats.Following {
		t.Fatalf("depois de deixar de seguir = %+v", tl.Stats)
	}
}
