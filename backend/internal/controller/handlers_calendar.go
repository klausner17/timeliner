package controller

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

type calendarEraJSON struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Abbr  string `json:"abbr"`
	Years *int   `json:"years"`
}

type calendarJSON struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Owner       string            `json:"owner"`
	Shared      bool              `json:"shared"`
	Eras        []calendarEraJSON `json:"eras"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

func toCalendar(v usecase.CalendarView) calendarJSON {
	c := v.Calendar
	eras := make([]calendarEraJSON, len(c.Eras))
	for i, e := range c.Eras {
		eras[i] = calendarEraJSON(e)
	}
	return calendarJSON{
		ID: c.ID, Name: c.Name, Description: c.Description, Owner: v.OwnerHandle, Shared: c.Shared, Eras: eras,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

type calendarFieldsJSON struct {
	Name        *string            `json:"name"`
	Description *string            `json:"description"`
	Shared      *bool              `json:"shared"`
	Eras        *[]calendarEraJSON `json:"eras"`
}

func (b calendarFieldsJSON) fields() usecase.CalendarFields {
	f := usecase.CalendarFields{Name: b.Name, Description: b.Description, Shared: b.Shared}
	if b.Eras != nil {
		eras := make([]domain.CalendarEra, len(*b.Eras))
		for i, e := range *b.Eras {
			eras[i] = domain.CalendarEra(e)
		}
		f.Eras = &eras
	}
	return f
}

func calendarRef(r *http.Request) usecase.CalendarRefInput {
	return usecase.CalendarRefInput{ActorID: actor(r), CalendarID: chi.URLParam(r, "id")}
}

func (c *Controller) listCalendars(w http.ResponseWriter, r *http.Request) {
	vs, err := c.uc.ListCalendars.Execute(r.Context(), usecase.ListCalendarsInput{ActorID: actor(r), Query: r.URL.Query().Get("q")})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	out := make([]calendarJSON, len(vs))
	for i, v := range vs {
		out[i] = toCalendar(v)
	}
	writeJSON(w, http.StatusOK, out)
}

func (c *Controller) createCalendar(w http.ResponseWriter, r *http.Request) {
	var body calendarFieldsJSON
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	v, err := c.uc.CreateCalendar.Execute(r.Context(), usecase.CreateCalendarInput{ActorID: actor(r), CalendarFields: body.fields()})
	c.writeCalendar(w, r, http.StatusCreated, v, err)
}

func (c *Controller) getCalendar(w http.ResponseWriter, r *http.Request) {
	v, err := c.uc.GetCalendar.Execute(r.Context(), calendarRef(r))
	c.writeCalendar(w, r, http.StatusOK, v, err)
}

func (c *Controller) updateCalendar(w http.ResponseWriter, r *http.Request) {
	var body calendarFieldsJSON
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	ref := calendarRef(r)
	v, err := c.uc.UpdateCalendar.Execute(r.Context(), usecase.UpdateCalendarInput{ActorID: ref.ActorID, CalendarID: ref.CalendarID, CalendarFields: body.fields()})
	c.writeCalendar(w, r, http.StatusOK, v, err)
}

func (c *Controller) deleteCalendar(w http.ResponseWriter, r *http.Request) {
	if _, err := c.uc.DeleteCalendar.Execute(r.Context(), calendarRef(r)); err != nil {
		c.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *Controller) writeCalendar(w http.ResponseWriter, r *http.Request, status int, v usecase.CalendarView, err error) {
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, status, toCalendar(v))
}
