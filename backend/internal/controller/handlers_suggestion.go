package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func (c *Controller) listSuggestions(w http.ResponseWriter, r *http.Request) {
	in := slugInput(r)
	vs, err := c.uc.ListSuggestions.Execute(r.Context(), usecase.ListSuggestionsInput{
		ActorID: in.ActorID, Slug: in.Slug, Status: domain.SuggestionStatus(r.URL.Query().Get("status")),
	})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, itemsJSON[suggestionJSON]{toSuggestions(vs)})
}

func (c *Controller) createSuggestion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind    domain.SuggestionKind `json:"kind"`
		EventID *string               `json:"eventId"`
		Event   *eventInputJSON       `json:"event"`
		Note    *string               `json:"note"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	fields := map[string]string{}
	if body.Event == nil {
		fields["event"] = "Preencha o evento."
	} else {
		body.Event.requireYear(fields, "event.")
	}
	if err := fieldsErr(fields); err != nil {
		c.fail(w, r, err)
		return
	}
	in := slugInput(r)
	v, err := c.uc.CreateSuggestion.Execute(r.Context(), usecase.CreateSuggestionInput{
		ActorID: in.ActorID, Slug: in.Slug, Kind: body.Kind, EventID: body.EventID, Event: body.Event.data(), Note: body.Note,
	})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toSuggestion(v))
}

func (c *Controller) resubmitSuggestion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Event *eventInputJSON `json:"event"`
		Note  *string         `json:"note"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	in := usecase.ResubmitSuggestionInput{ActorID: actor(r), SuggestionID: chi.URLParam(r, "id"), Note: body.Note}
	if body.Event != nil {
		fields := map[string]string{}
		body.Event.requireYear(fields, "event.")
		if err := fieldsErr(fields); err != nil {
			c.fail(w, r, err)
			return
		}
		data := body.Event.data()
		in.Event = &data
	}
	v, err := c.uc.ResubmitSuggestion.Execute(r.Context(), in)
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSuggestion(v))
}

func (c *Controller) reviewSuggestion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Decision usecase.ReviewDecision `json:"decision"`
		Feedback *string                `json:"feedback"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	v, err := c.uc.ReviewSuggestion.Execute(r.Context(), usecase.ReviewSuggestionInput{
		ActorID: actor(r), SuggestionID: chi.URLParam(r, "id"), Decision: body.Decision, Feedback: body.Feedback,
	})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSuggestion(v))
}

func (c *Controller) reopenSuggestion(w http.ResponseWriter, r *http.Request) {
	v, err := c.uc.ReopenSuggestion.Execute(r.Context(), usecase.SuggestionRefInput{ActorID: actor(r), SuggestionID: chi.URLParam(r, "id")})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSuggestion(v))
}
