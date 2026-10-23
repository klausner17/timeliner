package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

type timelineFieldsJSON struct {
	Title         *string                  `json:"title"`
	Description   *string                  `json:"description"`
	Category      *string                  `json:"category"`
	Color         *domain.TrackColor       `json:"color"`
	Visibility    *domain.Visibility       `json:"visibility"`
	Contributions *domain.ContributionMode `json:"contributions"`
	Draft         *bool                    `json:"draft"`
	Eras          *[]eraJSON               `json:"eras"`
	Tags          *[]string                `json:"tags"`
}

func (b timelineFieldsJSON) fields() usecase.TimelineFields {
	f := usecase.TimelineFields{
		Title: b.Title, Description: b.Description, Category: b.Category, Color: b.Color,
		Visibility: b.Visibility, Contributions: b.Contributions, Draft: b.Draft, Tags: b.Tags,
	}
	if b.Eras != nil {
		eras := make([]domain.Era, len(*b.Eras))
		for i, e := range *b.Eras {
			eras[i] = domain.Era(e)
		}
		f.Eras = &eras
	}
	return f
}

func slugInput(r *http.Request) usecase.SlugInput {
	return usecase.SlugInput{ActorID: actor(r), Slug: chi.URLParam(r, "slug")}
}

func (c *Controller) listTimelines(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fields := map[string]string{}
	in := usecase.ListTimelinesInput{
		ActorID: actor(r), Query: q.Get("q"), Category: q.Get("cat"), Tag: q.Get("tag"), OwnerHandle: q.Get("owner"),
		OnlyOpen: queryBool(r, "openToContributions", fields), Sort: usecase.TimelineSort(q.Get("sort")),
	}
	if p := queryInt(r, "page", fields); p != nil {
		in.Page = *p
	}
	if p := queryInt(r, "pageSize", fields); p != nil {
		in.PageSize = *p
	}
	if err := fieldsErr(fields); err != nil {
		c.fail(w, r, err)
		return
	}
	page, err := c.uc.ListTimelines.Execute(r.Context(), in)
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, timelinePageJSON{Items: toTimelines(page.Items), Page: page.Page, PageSize: page.PageSize, Total: page.Total})
}

func (c *Controller) createTimeline(w http.ResponseWriter, r *http.Request) {
	var body struct {
		timelineFieldsJSON
		Events []eventInputJSON `json:"events"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	fields := map[string]string{}
	events := make([]domain.EventData, len(body.Events))
	for i, e := range body.Events {
		e.requireYear(fields, fmt.Sprintf("events[%d].", i))
		events[i] = e.data()
	}
	if err := fieldsErr(fields); err != nil {
		c.fail(w, r, err)
		return
	}
	v, err := c.uc.CreateTimeline.Execute(r.Context(), usecase.CreateTimelineInput{ActorID: actor(r), TimelineFields: body.fields(), Events: events})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toTimelineDetail(v))
}

func (c *Controller) getTimeline(w http.ResponseWriter, r *http.Request) {
	v, err := c.uc.GetTimeline.Execute(r.Context(), slugInput(r))
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toTimelineDetail(v))
}

func (c *Controller) updateTimeline(w http.ResponseWriter, r *http.Request) {
	var body timelineFieldsJSON
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	in := slugInput(r)
	v, err := c.uc.UpdateTimeline.Execute(r.Context(), usecase.UpdateTimelineInput{ActorID: in.ActorID, Slug: in.Slug, TimelineFields: body.fields()})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toTimelineDetail(v))
}

func (c *Controller) deleteTimeline(w http.ResponseWriter, r *http.Request) {
	if _, err := c.uc.DeleteTimeline.Execute(r.Context(), slugInput(r)); err != nil {
		c.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *Controller) followTimeline(follow bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in := slugInput(r)
		if _, err := c.uc.FollowTimeline.Execute(r.Context(), usecase.FollowTimelineInput{ActorID: in.ActorID, Slug: in.Slug, Follow: follow}); err != nil {
			c.fail(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (c *Controller) listTags(w http.ResponseWriter, r *http.Request) {
	tags, err := c.uc.ListTags.Execute(r.Context(), usecase.Empty{})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	out := make([]tagJSON, len(tags))
	for i, t := range tags {
		out[i] = tagJSON(t)
	}
	writeJSON(w, http.StatusOK, itemsJSON[tagJSON]{out})
}

func (c *Controller) listEvents(w http.ResponseWriter, r *http.Request) {
	fields := map[string]string{}
	from, to := queryInt(r, "from", fields), queryInt(r, "to", fields)
	if err := fieldsErr(fields); err != nil {
		c.fail(w, r, err)
		return
	}
	in := slugInput(r)
	vs, err := c.uc.ListEvents.Execute(r.Context(), usecase.ListEventsInput{ActorID: in.ActorID, Slug: in.Slug, From: from, To: to})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, itemsJSON[eventJSON]{toEvents(vs)})
}

func (c *Controller) createEvent(w http.ResponseWriter, r *http.Request) {
	var body eventInputJSON
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	fields := map[string]string{}
	body.requireYear(fields, "")
	if err := fieldsErr(fields); err != nil {
		c.fail(w, r, err)
		return
	}
	in := slugInput(r)
	v, err := c.uc.CreateEvent.Execute(r.Context(), usecase.CreateEventInput{ActorID: in.ActorID, Slug: in.Slug, Event: body.data()})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toEvent(v))
}

func (c *Controller) updateEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Year        *int             `json:"year"`
		EndYear     Optional[int]    `json:"endYear"`
		Date        Optional[string] `json:"date"`
		Title       *string          `json:"title"`
		Description *string          `json:"description"`
		SourceURL   Optional[string] `json:"sourceUrl"`
		Era         Optional[string] `json:"era"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	in := slugInput(r)
	v, err := c.uc.UpdateEvent.Execute(r.Context(), usecase.UpdateEventInput{
		ActorID: in.ActorID, Slug: in.Slug, EventID: chi.URLParam(r, "eventId"),
		Patch: usecase.EventPatch{
			Year: body.Year, EndYear: body.EndYear.ptr(), Date: body.Date.ptr(), Title: body.Title,
			Description: body.Description, SourceURL: body.SourceURL.ptr(), Era: body.Era.ptr(),
		},
	})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toEvent(v))
}

func (c *Controller) deleteEvent(w http.ResponseWriter, r *http.Request) {
	in := slugInput(r)
	if _, err := c.uc.DeleteEvent.Execute(r.Context(), usecase.EventRefInput{ActorID: in.ActorID, Slug: in.Slug, EventID: chi.URLParam(r, "eventId")}); err != nil {
		c.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *Controller) compare(w http.ResponseWriter, r *http.Request) {
	fields := map[string]string{}
	start, end := queryInt(r, "start", fields), queryInt(r, "end", fields)
	if err := fieldsErr(fields); err != nil {
		c.fail(w, r, err)
		return
	}
	slugs := strings.Split(r.URL.Query().Get("slugs"), ",")
	v, err := c.uc.Compare.Execute(r.Context(), usecase.CompareInput{ActorID: actor(r), Slugs: slugs, Start: start, End: end})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toComparison(v))
}

func (c *Controller) listSavedComparisons(w http.ResponseWriter, r *http.Request) {
	cs, err := c.uc.ListSavedComparisons.Execute(r.Context(), usecase.ActorInput{ActorID: actor(r)})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	out := make([]savedComparisonJSON, len(cs))
	for i, s := range cs {
		out[i] = toSavedComparison(s)
	}
	writeJSON(w, http.StatusOK, itemsJSON[savedComparisonJSON]{out})
}

func (c *Controller) saveComparison(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Timelines []string `json:"timelines"`
		Start     *int     `json:"start"`
		End       *int     `json:"end"`
	}
	if err := decode(r, &body); err != nil {
		c.fail(w, r, err)
		return
	}
	fields := map[string]string{}
	if body.Start == nil {
		fields["start"] = "Informe o ano inicial."
	}
	if body.End == nil {
		fields["end"] = "Informe o ano final."
	}
	if err := fieldsErr(fields); err != nil {
		c.fail(w, r, err)
		return
	}
	s, err := c.uc.SaveComparison.Execute(r.Context(), usecase.SaveComparisonInput{ActorID: actor(r), Slugs: body.Timelines, Start: *body.Start, End: *body.End})
	if err != nil {
		c.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toSavedComparison(s))
}

func (c *Controller) deleteSavedComparison(w http.ResponseWriter, r *http.Request) {
	if _, err := c.uc.DeleteSavedComparison.Execute(r.Context(), usecase.DeleteSavedComparisonInput{ActorID: actor(r), ComparisonID: chi.URLParam(r, "id")}); err != nil {
		c.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
