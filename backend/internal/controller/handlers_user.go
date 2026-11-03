package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func (c *Controller) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Handle   string `json:"handle"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	out, err := c.uc.Register.Execute(r.Context(), usecase.RegisterInput(body))
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, authJSON{Token: out.Token, User: toUser(out.User)})
}

func (c *Controller) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	out, err := c.uc.Login.Execute(r.Context(), usecase.LoginInput(body))
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, authJSON{Token: out.Token, User: toUser(out.User)})
}

func (c *Controller) getMe(w http.ResponseWriter, r *http.Request) {
	u, err := c.uc.GetMe.Execute(r.Context(), usecase.ActorInput{ActorID: actor(r)})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUser(u))
}

func (c *Controller) updateMe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name *string      `json:"name"`
		Bio  *string      `json:"bio"`
		Tint *domain.Tint `json:"tint"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	u, err := c.uc.UpdateProfile.Execute(r.Context(), usecase.UpdateProfileInput{ActorID: actor(r), Name: body.Name, Bio: body.Bio, Tint: body.Tint})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUser(u))
}

func handleInput(r *http.Request) usecase.HandleInput {
	return usecase.HandleInput{ActorID: actor(r), Handle: chi.URLParam(r, "handle")}
}

func (c *Controller) getProfile(w http.ResponseWriter, r *http.Request) {
	v, err := c.uc.GetProfile.Execute(r.Context(), handleInput(r))
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProfile(v))
}

func (c *Controller) listContributions(w http.ResponseWriter, r *http.Request) {
	h := handleInput(r)
	fields := map[string]string{}
	in := usecase.ListContributionsInput{ActorID: h.ActorID, Handle: h.Handle, Slug: r.URL.Query().Get("timeline"), PageInput: pageInput(r, fields)}
	if err := fieldsErr(fields); err != nil {
		c.fail(w, r, err)
		return
	}
	page, err := c.uc.ListContributions.Execute(r.Context(), in)
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPage(page, toSuggestions(page.Items)))
}

func (c *Controller) followUser(follow bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := c.uc.FollowUser.Execute(r.Context(), usecase.FollowUserInput{ActorID: actor(r), Handle: chi.URLParam(r, "handle"), Follow: follow})
		if err != nil {
			c.fail(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (c *Controller) listFollowing(w http.ResponseWriter, r *http.Request) {
	h := handleInput(r)
	fields := map[string]string{}
	in := usecase.ListFollowingInput{ActorID: h.ActorID, Handle: h.Handle, PageInput: pageInput(r, fields)}
	if err := fieldsErr(fields); err != nil {
		c.fail(w, r, err)
		return
	}
	v, err := c.uc.ListFollowing.Execute(r.Context(), in)
	if err != nil {
		c.fail(w, r, err)
		return
	}
	out := followingJSON{
		Users: make([]userSummaryJSON, len(v.Users)), Timelines: toTimelines(v.Timelines),
		UsersTotal: v.UsersTotal, TimelinesTotal: v.TimelinesTotal, Page: v.Page, PageSize: v.PageSize,
	}
	for i, u := range v.Users {
		out.Users[i] = toUserSummary(u)
	}
	writeJSON(w, http.StatusOK, out)
}
