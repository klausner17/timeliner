package controller

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

// UseCases são os casos de uso que o controller chama, pelo contrato UseCase[I, O].
type UseCases struct {
	Register     usecase.UseCase[usecase.RegisterInput, usecase.AuthOutput]
	Login        usecase.UseCase[usecase.LoginInput, usecase.AuthOutput]
	Authenticate usecase.UseCase[string, string]

	GetMe             usecase.UseCase[usecase.ActorInput, domain.User]
	UpdateProfile     usecase.UseCase[usecase.UpdateProfileInput, domain.User]
	GetProfile        usecase.UseCase[usecase.HandleInput, usecase.UserView]
	ListContributions usecase.UseCase[usecase.HandleInput, []usecase.SuggestionView]
	FollowUser        usecase.UseCase[usecase.FollowUserInput, usecase.Empty]
	ListFollowing     usecase.UseCase[usecase.HandleInput, usecase.FollowingView]

	ListTimelines  usecase.UseCase[usecase.ListTimelinesInput, usecase.TimelinePage]
	CreateTimeline usecase.UseCase[usecase.CreateTimelineInput, usecase.TimelineDetailView]
	GetTimeline    usecase.UseCase[usecase.SlugInput, usecase.TimelineDetailView]
	UpdateTimeline usecase.UseCase[usecase.UpdateTimelineInput, usecase.TimelineDetailView]
	DeleteTimeline usecase.UseCase[usecase.SlugInput, usecase.Empty]
	FollowTimeline usecase.UseCase[usecase.FollowTimelineInput, usecase.Empty]
	ListTags       usecase.UseCase[usecase.Empty, []domain.TagCount]

	ListEvents      usecase.UseCase[usecase.ListEventsInput, usecase.EventPage]
	SummarizeEvents usecase.UseCase[usecase.SummarizeEventsInput, usecase.EventWindow]
	CreateEvent     usecase.UseCase[usecase.CreateEventInput, usecase.EventView]
	UpdateEvent     usecase.UseCase[usecase.UpdateEventInput, usecase.EventView]
	DeleteEvent     usecase.UseCase[usecase.EventRefInput, usecase.Empty]

	ListSuggestions    usecase.UseCase[usecase.ListSuggestionsInput, []usecase.SuggestionView]
	CreateSuggestion   usecase.UseCase[usecase.CreateSuggestionInput, usecase.SuggestionView]
	ResubmitSuggestion usecase.UseCase[usecase.ResubmitSuggestionInput, usecase.SuggestionView]
	ReviewSuggestion   usecase.UseCase[usecase.ReviewSuggestionInput, usecase.SuggestionView]
	ReopenSuggestion   usecase.UseCase[usecase.SuggestionRefInput, usecase.SuggestionView]

	Compare               usecase.UseCase[usecase.CompareInput, usecase.ComparisonView]
	SaveComparison        usecase.UseCase[usecase.SaveComparisonInput, domain.SavedComparison]
	ListSavedComparisons  usecase.UseCase[usecase.ActorInput, []domain.SavedComparison]
	DeleteSavedComparison usecase.UseCase[usecase.DeleteSavedComparisonInput, usecase.Empty]
}

// UseCasesFrom monta todos os casos de uso sobre as mesmas dependências.
func UseCasesFrom(d *usecase.Deps) UseCases {
	return UseCases{
		Register: usecase.NewRegister(d), Login: usecase.NewLogin(d), Authenticate: usecase.NewAuthenticate(d),
		GetMe: usecase.NewGetMe(d), UpdateProfile: usecase.NewUpdateProfile(d), GetProfile: usecase.NewGetProfile(d),
		ListContributions: usecase.NewListContributions(d), FollowUser: usecase.NewFollowUser(d), ListFollowing: usecase.NewListFollowing(d),
		ListTimelines: usecase.NewListTimelines(d), CreateTimeline: usecase.NewCreateTimeline(d), GetTimeline: usecase.NewGetTimeline(d),
		UpdateTimeline: usecase.NewUpdateTimeline(d), DeleteTimeline: usecase.NewDeleteTimeline(d),
		FollowTimeline: usecase.NewFollowTimeline(d), ListTags: usecase.NewListTags(d),
		ListEvents: usecase.NewListEvents(d), SummarizeEvents: usecase.NewSummarizeEvents(d), CreateEvent: usecase.NewCreateEvent(d), UpdateEvent: usecase.NewUpdateEvent(d),
		DeleteEvent:     usecase.NewDeleteEvent(d),
		ListSuggestions: usecase.NewListSuggestions(d), CreateSuggestion: usecase.NewCreateSuggestion(d),
		ResubmitSuggestion: usecase.NewResubmitSuggestion(d), ReviewSuggestion: usecase.NewReviewSuggestion(d),
		ReopenSuggestion: usecase.NewReopenSuggestion(d),
		Compare:          usecase.NewCompare(d), SaveComparison: usecase.NewSaveComparison(d),
		ListSavedComparisons: usecase.NewListSavedComparisons(d), DeleteSavedComparison: usecase.NewDeleteSavedComparison(d),
	}
}

type Controller struct {
	uc     UseCases
	log    *slog.Logger
	health func(context.Context) error
}

// New devolve o handler com todas as rotas sob /api. health confere as dependências
// (ex.: ping no banco) e pode ser nil.
func New(uc UseCases, log *slog.Logger, health func(context.Context) error) http.Handler {
	c := &Controller{uc: uc, log: log, health: health}
	r := chi.NewRouter()
	r.Use(c.recoverer)
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "Endereço não encontrado.", nil)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Operação não permitida neste endereço.", nil)
	})
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", c.getHealth)
		r.Group(func(r chi.Router) {
			r.Use(c.authenticate)
			c.routes(r)
		})
	})
	return r
}

func (c *Controller) routes(r chi.Router) {
	r.Post("/auth/register", c.register)
	r.Post("/auth/login", c.login)
	r.Get("/me", c.getMe)
	r.Patch("/me", c.updateMe)
	r.Get("/me/comparisons", c.listSavedComparisons)
	r.Post("/me/comparisons", c.saveComparison)
	r.Delete("/me/comparisons/{id}", c.deleteSavedComparison)

	r.Get("/timelines", c.listTimelines)
	r.Post("/timelines", c.createTimeline)
	r.Get("/timelines/{slug}", c.getTimeline)
	r.Patch("/timelines/{slug}", c.updateTimeline)
	r.Delete("/timelines/{slug}", c.deleteTimeline)
	r.Post("/timelines/{slug}/follow", c.followTimeline(true))
	r.Delete("/timelines/{slug}/follow", c.followTimeline(false))
	r.Get("/timelines/{slug}/events", c.listEvents)
	r.Get("/timelines/{slug}/summary", c.summarizeEvents)
	r.Post("/timelines/{slug}/events", c.createEvent)
	r.Patch("/timelines/{slug}/events/{eventId}", c.updateEvent)
	r.Delete("/timelines/{slug}/events/{eventId}", c.deleteEvent)
	r.Get("/timelines/{slug}/suggestions", c.listSuggestions)
	r.Post("/timelines/{slug}/suggestions", c.createSuggestion)

	r.Patch("/suggestions/{id}", c.resubmitSuggestion)
	r.Post("/suggestions/{id}/review", c.reviewSuggestion)
	r.Post("/suggestions/{id}/reopen", c.reopenSuggestion)

	r.Get("/compare", c.compare)
	r.Get("/tags", c.listTags)

	r.Get("/users/{handle}", c.getProfile)
	r.Post("/users/{handle}/follow", c.followUser(true))
	r.Delete("/users/{handle}/follow", c.followUser(false))
	r.Get("/users/{handle}/following", c.listFollowing)
	r.Get("/users/{handle}/contributions", c.listContributions)
}

func (c *Controller) getHealth(w http.ResponseWriter, r *http.Request) {
	if c.health != nil {
		if err := c.health(r.Context()); err != nil {
			c.log.ErrorContext(r.Context(), "health falhou", "err", err)
			writeError(w, http.StatusServiceUnavailable, "unavailable", "Serviço indisponível no momento.", nil)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
