package usecase_test

import (
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

type suggestionWorld struct {
	d          *usecase.Deps
	owner, bob domain.User
	timeline   usecase.TimelineDetailView
}

func newSuggestionWorld(t *testing.T) suggestionWorld {
	d := newDeps()
	owner, bob := register(t, d, "ana"), register(t, d, "beto")
	tl := createTimeline(t, d, owner.ID, "Escândalos", ev(1992, "Impeachment"))
	return suggestionWorld{d, owner, bob, tl}
}

func (w suggestionWorld) suggest(t *testing.T, in usecase.CreateSuggestionInput) usecase.SuggestionView {
	t.Helper()
	in.ActorID, in.Slug = w.bob.ID, w.timeline.Timeline.Slug
	return must[usecase.SuggestionView](t)(usecase.NewCreateSuggestion(w.d).Execute(ctx, in))
}

func (w suggestionWorld) review(t *testing.T, id string, decision usecase.ReviewDecision, feedback *string) (usecase.SuggestionView, error) {
	return usecase.NewReviewSuggestion(w.d).Execute(ctx, usecase.ReviewSuggestionInput{ActorID: w.owner.ID, SuggestionID: id, Decision: decision, Feedback: feedback})
}

func (w suggestionWorld) events(t *testing.T) []usecase.EventView {
	t.Helper()
	return must[[]usecase.EventView](t)(usecase.NewListEvents(w.d).Execute(ctx, usecase.ListEventsInput{Slug: w.timeline.Timeline.Slug}))
}

func TestApproveNewEventAndReopen(t *testing.T) {
	w := newSuggestionWorld(t)
	s := w.suggest(t, usecase.CreateSuggestionInput{Kind: domain.SuggestionNewEvent, Event: ev(1989, "Eleição"), Note: ptr("fonte no link")})
	if s.Suggestion.Status != domain.StatusPending || s.AuthorHandle != "beto" || s.Timeline.Slug != "escandalos" {
		t.Fatalf("sugestão = %+v", s)
	}
	detail := must[usecase.TimelineDetailView](t)(usecase.NewGetTimeline(w.d).Execute(ctx, usecase.SlugInput{Slug: "escandalos"}))
	if detail.Stats.PendingSuggestions != 1 {
		t.Fatalf("pendentes = %d", detail.Stats.PendingSuggestions)
	}

	approved := must[usecase.SuggestionView](t)(w.review(t, s.Suggestion.ID, usecase.DecisionApprove, nil))
	if approved.Suggestion.Status != domain.StatusApproved || approved.Suggestion.AppliedEventID == nil {
		t.Fatalf("aprovada = %+v", approved.Suggestion)
	}
	evs := w.events(t)
	if len(evs) != 2 || evs[0].Event.Title != "Eleição" || evs[0].AuthorHandle != "beto" {
		t.Fatalf("eventos = %+v", evs)
	}
	_, err := w.review(t, s.Suggestion.ID, usecase.DecisionReject, nil)
	wantKind(t, err, domain.KindConflict)

	detail = must[usecase.TimelineDetailView](t)(usecase.NewGetTimeline(w.d).Execute(ctx, usecase.SlugInput{Slug: "escandalos"}))
	if detail.Stats.ContributorCount != 1 {
		t.Fatalf("colaboradores = %d", detail.Stats.ContributorCount)
	}
	profile := must[usecase.UserView](t)(usecase.NewGetProfile(w.d).Execute(ctx, usecase.HandleInput{Handle: "beto"}))
	if profile.Stats.ApprovedContributions != 1 || profile.Stats.Events != 1 {
		t.Fatalf("perfil = %+v", profile.Stats)
	}

	reopened := must[usecase.SuggestionView](t)(usecase.NewReopenSuggestion(w.d).Execute(ctx, usecase.SuggestionRefInput{ActorID: w.owner.ID, SuggestionID: s.Suggestion.ID}))
	if reopened.Suggestion.Status != domain.StatusPending || reopened.Suggestion.AppliedEventID != nil {
		t.Fatalf("reaberta = %+v", reopened.Suggestion)
	}
	if evs := w.events(t); len(evs) != 1 {
		t.Fatalf("o evento criado devia sumir: %+v", evs)
	}
}

func TestApproveFixKeepsPreviousAndReopenRestores(t *testing.T) {
	w := newSuggestionWorld(t)
	original := w.timeline.Events[0].Event
	s := w.suggest(t, usecase.CreateSuggestionInput{
		Kind: domain.SuggestionDateFix, EventID: &original.ID,
		Event: domain.EventData{Year: 1992, Date: ptr("29 de setembro de 1992"), Title: "Impeachment"},
	})
	approved := must[usecase.SuggestionView](t)(w.review(t, s.Suggestion.ID, usecase.DecisionApprove, nil))
	if approved.Previous == nil || approved.Previous.Event.Date != nil || approved.Previous.AuthorHandle != "ana" {
		t.Fatalf("previous = %+v", approved.Previous)
	}
	evs := w.events(t)
	if *evs[0].Event.Date != "29 de setembro de 1992" || evs[0].AuthorHandle != "ana" {
		t.Fatalf("evento corrigido = %+v", evs[0])
	}

	must[usecase.SuggestionView](t)(usecase.NewReopenSuggestion(w.d).Execute(ctx, usecase.SuggestionRefInput{ActorID: w.owner.ID, SuggestionID: s.Suggestion.ID}))
	if evs := w.events(t); evs[0].Event.Date != nil {
		t.Fatalf("devia voltar ao original: %+v", evs[0])
	}
}

func TestRequestChangesAndResubmit(t *testing.T) {
	w := newSuggestionWorld(t)
	s := w.suggest(t, usecase.CreateSuggestionInput{Kind: domain.SuggestionNewEvent, Event: ev(2005, "Mensalão")})

	_, err := w.review(t, s.Suggestion.ID, usecase.DecisionRequestChanges, nil)
	wantKind(t, err, domain.KindValidation)
	asked := must[usecase.SuggestionView](t)(w.review(t, s.Suggestion.ID, usecase.DecisionRequestChanges, ptr("Coloque a fonte")))
	if asked.Suggestion.Status != domain.StatusChangesRequested || *asked.Suggestion.Feedback != "Coloque a fonte" {
		t.Fatalf("ajuste = %+v", asked.Suggestion)
	}

	resubmit := usecase.NewResubmitSuggestion(w.d)
	_, err = resubmit.Execute(ctx, usecase.ResubmitSuggestionInput{ActorID: w.owner.ID, SuggestionID: s.Suggestion.ID})
	wantKind(t, err, domain.KindForbidden)
	again := must[usecase.SuggestionView](t)(resubmit.Execute(ctx, usecase.ResubmitSuggestionInput{
		ActorID: w.bob.ID, SuggestionID: s.Suggestion.ID,
		Event: &domain.EventData{Year: 2005, Title: "Mensalão", SourceURL: ptr("https://exemplo.com/mensalao")},
	}))
	if again.Suggestion.Status != domain.StatusPending || again.Suggestion.Proposed.SourceURL == nil {
		t.Fatalf("reenviada = %+v", again.Suggestion)
	}
	rejected := must[usecase.SuggestionView](t)(w.review(t, s.Suggestion.ID, usecase.DecisionReject, nil))
	if rejected.Suggestion.Status != domain.StatusRejected {
		t.Fatalf("recusada = %+v", rejected.Suggestion)
	}
	_, err = resubmit.Execute(ctx, usecase.ResubmitSuggestionInput{ActorID: w.bob.ID, SuggestionID: s.Suggestion.ID})
	wantKind(t, err, domain.KindConflict)

	list := must[[]usecase.SuggestionView](t)(usecase.NewListSuggestions(w.d).Execute(ctx, usecase.ListSuggestionsInput{Slug: "escandalos", Status: domain.StatusRejected}))
	if len(list) != 1 {
		t.Fatalf("recusadas = %d", len(list))
	}
	contrib := must[[]usecase.SuggestionView](t)(usecase.NewListContributions(w.d).Execute(ctx, usecase.HandleInput{Handle: "beto"}))
	if len(contrib) != 1 {
		t.Fatalf("contribuições = %d", len(contrib))
	}
}

func TestSuggestionRules(t *testing.T) {
	w := newSuggestionWorld(t)
	create := usecase.NewCreateSuggestion(w.d)
	base := usecase.CreateSuggestionInput{ActorID: w.bob.ID, Slug: "escandalos", Kind: domain.SuggestionDescriptionFix, Event: ev(1992, "x")}

	_, err := create.Execute(ctx, base)
	wantKind(t, err, domain.KindValidation)
	base.EventID = ptr("ev_nao_existe")
	_, err = create.Execute(ctx, base)
	wantKind(t, err, domain.KindValidation)

	anon := base
	anon.ActorID = ""
	_, err = create.Execute(ctx, anon)
	wantKind(t, err, domain.KindUnauthorized)

	must[usecase.TimelineDetailView](t)(usecase.NewUpdateTimeline(w.d).Execute(ctx, usecase.UpdateTimelineInput{
		ActorID: w.owner.ID, Slug: "escandalos", TimelineFields: usecase.TimelineFields{Contributions: ptr(domain.ContributionsClosed)},
	}))
	base.Kind, base.EventID = domain.SuggestionNewEvent, nil
	_, err = create.Execute(ctx, base)
	wantKind(t, err, domain.KindForbidden)

	must[usecase.TimelineDetailView](t)(usecase.NewUpdateTimeline(w.d).Execute(ctx, usecase.UpdateTimelineInput{
		ActorID: w.owner.ID, Slug: "escandalos", TimelineFields: usecase.TimelineFields{Contributions: ptr(domain.ContributionsSuggestions)},
	}))
	s := must[usecase.SuggestionView](t)(create.Execute(ctx, base))
	_, err = usecase.NewReviewSuggestion(w.d).Execute(ctx, usecase.ReviewSuggestionInput{ActorID: w.bob.ID, SuggestionID: s.Suggestion.ID, Decision: usecase.DecisionApprove})
	wantKind(t, err, domain.KindForbidden)
	_, err = usecase.NewReopenSuggestion(w.d).Execute(ctx, usecase.SuggestionRefInput{ActorID: w.owner.ID, SuggestionID: s.Suggestion.ID})
	wantKind(t, err, domain.KindConflict)
}
