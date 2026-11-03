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
		// Website é um campo-isca: fica escondido no formulário e só robô preenche.
		Website string `json:"website"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	if body.Website != "" {
		c.fail(w, r, &domain.Error{Kind: domain.KindValidation, Message: "Não foi possível criar a conta."})
		return
	}
	out, err := c.uc.Register.Execute(r.Context(), usecase.RegisterInput{
		Handle: body.Handle, Name: body.Name, Email: body.Email, Password: body.Password, ClientIP: c.clientIP(r),
	})
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
	out, err := c.uc.Login.Execute(r.Context(), usecase.LoginInput{Email: body.Email, Password: body.Password, ClientIP: c.clientIP(r)})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, authJSON{Token: out.Token, User: toUser(out.User)})
}

func (c *Controller) logout(w http.ResponseWriter, r *http.Request) {
	if _, err := c.uc.Logout.Execute(r.Context(), bearer(r)); err != nil {
		c.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *Controller) signupStatus(w http.ResponseWriter, r *http.Request) {
	s, err := c.uc.SignupStatus.Execute(r.Context(), usecase.Empty{})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"open": s.Open})
}

func (c *Controller) changePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	_, err := c.uc.ChangePassword.Execute(r.Context(), usecase.ChangePasswordInput{
		ActorID: actor(r), Token: bearer(r), CurrentPassword: body.CurrentPassword, NewPassword: body.NewPassword,
	})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *Controller) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	u, err := c.uc.VerifyEmail.Execute(r.Context(), body.Token)
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUser(u))
}

func (c *Controller) resendVerification(w http.ResponseWriter, r *http.Request) {
	if _, err := c.uc.ResendVerification.Execute(r.Context(), usecase.ActorInput{ActorID: actor(r)}); err != nil {
		c.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *Controller) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	if _, err := c.uc.RequestPasswordReset.Execute(r.Context(), usecase.RequestPasswordResetInput{Email: body.Email, ClientIP: c.clientIP(r)}); err != nil {
		c.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *Controller) confirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	out, err := c.uc.ConfirmPasswordReset.Execute(r.Context(), usecase.ConfirmPasswordResetInput{Token: body.Token, Password: body.Password})
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
