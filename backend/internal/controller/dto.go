package controller

import (
	"time"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

// Os tipos deste arquivo são o JSON de api/openapi.yaml. As funções to* convertem
// o Output dos usecases para eles.

type userSummaryJSON struct {
	Handle string      `json:"handle"`
	Name   string      `json:"name"`
	Tint   domain.Tint `json:"tint"`
}

func toUserSummary(u domain.User) userSummaryJSON {
	return userSummaryJSON{Handle: u.Handle, Name: u.Name, Tint: u.Tint}
}

type userJSON struct {
	userSummaryJSON
	Email       string    `json:"email"`
	Bio         string    `json:"bio"`
	MemberSince time.Time `json:"memberSince"`
}

func toUser(u domain.User) userJSON {
	return userJSON{userSummaryJSON: toUserSummary(u), Email: u.Email, Bio: u.Bio, MemberSince: u.CreatedAt}
}

type profileStatsJSON struct {
	Timelines             int `json:"timelines"`
	Events                int `json:"events"`
	ApprovedContributions int `json:"approvedContributions"`
	Followers             int `json:"followers"`
}

type profileJSON struct {
	userSummaryJSON
	Bio         string           `json:"bio"`
	MemberSince time.Time        `json:"memberSince"`
	Following   bool             `json:"following"`
	Stats       profileStatsJSON `json:"stats"`
}

func toProfile(v usecase.UserView) profileJSON {
	return profileJSON{
		userSummaryJSON: toUserSummary(v.User), Bio: v.User.Bio, MemberSince: v.User.CreatedAt, Following: v.Stats.Following,
		Stats: profileStatsJSON{v.Stats.Timelines, v.Stats.Events, v.Stats.ApprovedContributions, v.Stats.Followers},
	}
}

type authJSON struct {
	Token string   `json:"token"`
	User  userJSON `json:"user"`
}

type eraJSON struct {
	Name       string              `json:"name"`
	Start      int                 `json:"start"`
	End        *int                `json:"end"`
	Color      *domain.PeriodColor `json:"color"`
	StartMonth *int                `json:"startMonth"`
	StartDay   *int                `json:"startDay"`
	EndMonth   *int                `json:"endMonth"`
	EndDay     *int                `json:"endDay"`
}

// timelineCalendarJSON é o calendário que vem dentro da timeline: o bastante para a
// tela escrever os anos ("T.A. 3019").
type timelineCalendarJSON struct {
	ID   string            `json:"id"`
	Name string            `json:"name"`
	Eras []calendarEraJSON `json:"eras"`
}

type timelineJSON struct {
	Slug               string                  `json:"slug"`
	Title              string                  `json:"title"`
	Description        string                  `json:"description"`
	Category           string                  `json:"category"`
	Color              domain.TrackColor       `json:"color"`
	Owner              userSummaryJSON         `json:"owner"`
	Visibility         domain.Visibility       `json:"visibility"`
	Contributions      domain.ContributionMode `json:"contributions"`
	Draft              bool                    `json:"draft"`
	Eras               []eraJSON               `json:"eras"`
	Tags               []string                `json:"tags"`
	Calendar           *timelineCalendarJSON   `json:"calendar"`
	StartYear          *int                    `json:"startYear"`
	EndYear            *int                    `json:"endYear"`
	EventCount         int                     `json:"eventCount"`
	ContributorCount   int                     `json:"contributorCount"`
	PendingSuggestions int                     `json:"pendingSuggestions"`
	Followers          int                     `json:"followers"`
	Following          bool                    `json:"following"`
	Points             []float64               `json:"points"`
	CreatedAt          time.Time               `json:"createdAt"`
	UpdatedAt          time.Time               `json:"updatedAt"`
}

func toTimeline(v usecase.TimelineView) timelineJSON {
	t := v.Timeline
	eras := make([]eraJSON, len(t.Eras))
	for i, e := range t.Eras {
		eras[i] = eraJSON(e)
	}
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}
	var cal *timelineCalendarJSON
	if v.Calendar != nil {
		eras := make([]calendarEraJSON, len(v.Calendar.Eras))
		for i, e := range v.Calendar.Eras {
			eras[i] = calendarEraJSON(e)
		}
		cal = &timelineCalendarJSON{ID: v.Calendar.ID, Name: v.Calendar.Name, Eras: eras}
	}
	return timelineJSON{
		Calendar: cal,
		Slug:     t.Slug, Title: t.Title, Description: t.Description, Category: t.Category, Color: t.Color,
		Owner: toUserSummary(v.Owner), Visibility: t.Visibility, Contributions: t.Contributions, Draft: t.Draft,
		Eras: eras, Tags: tags, StartYear: v.Stats.StartYear, EndYear: v.Stats.EndYear,
		EventCount: v.Stats.EventCount, ContributorCount: v.Stats.ContributorCount,
		PendingSuggestions: v.Stats.PendingSuggestions, Followers: v.Stats.Followers, Following: v.Stats.Following,
		Points: v.Points, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toTimelines(vs []usecase.TimelineView) []timelineJSON {
	out := make([]timelineJSON, len(vs))
	for i, v := range vs {
		out[i] = toTimeline(v)
	}
	return out
}

type timelineDetailJSON struct {
	timelineJSON
	// Events é só a primeira página; o resto vem de /timelines/{slug}/events?cursor=.
	Events           []eventJSON `json:"events"`
	EventsNextCursor *string     `json:"eventsNextCursor"`
}

func toTimelineDetail(v usecase.TimelineDetailView) timelineDetailJSON {
	return timelineDetailJSON{timelineJSON: toTimeline(v.TimelineView), Events: toEvents(v.Events.Items), EventsNextCursor: v.Events.NextCursor}
}

type timelinePageJSON struct {
	Items    []timelineJSON `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	Total    int            `json:"total"`
}

// eventInputJSON é o EventInput do contrato, usado na entrada e no "event" das sugestões.
type eventInputJSON struct {
	Year        *int    `json:"year"`
	Month       *int    `json:"month"`
	Day         *int    `json:"day"`
	Time        *string `json:"time"`
	EndYear     *int    `json:"endYear"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	SourceURL   *string `json:"sourceUrl"`
	Era         *string `json:"era"`
}

func (e eventInputJSON) data() domain.EventData {
	d := domain.EventData{Month: e.Month, Day: e.Day, Time: e.Time, EndYear: e.EndYear, Title: e.Title, Description: e.Description, SourceURL: e.SourceURL, Era: e.Era}
	if e.Year != nil {
		d.Year = *e.Year
	}
	return d
}

// requireYear marca year como obrigatório, já que 0 é um ano válido e não dá para
// distinguir de "não veio".
func (e eventInputJSON) requireYear(fields map[string]string, prefix string) {
	if e.Year == nil {
		fields[prefix+"year"] = "Informe o ano."
	}
}

func toEventInput(d domain.EventData) eventInputJSON {
	y := d.Year
	return eventInputJSON{Year: &y, Month: d.Month, Day: d.Day, Time: d.Time, EndYear: d.EndYear, Title: d.Title, Description: d.Description, SourceURL: d.SourceURL, Era: d.Era}
}

type eventJSON struct {
	ID          string    `json:"id"`
	Year        int       `json:"year"`
	Month       *int      `json:"month"`
	Day         *int      `json:"day"`
	Time        *string   `json:"time"`
	EndYear     *int      `json:"endYear"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	SourceURL   *string   `json:"sourceUrl"`
	By          string    `json:"by"`
	Era         *string   `json:"era"`
	CreatedAt   time.Time `json:"createdAt"`
}

func toEvent(v usecase.EventView) eventJSON {
	e := v.Event
	return eventJSON{
		ID: e.ID, Year: e.Year, Month: e.Month, Day: e.Day, Time: e.Time, EndYear: e.EndYear, Title: e.Title, Description: e.Description,
		SourceURL: e.SourceURL, By: v.AuthorHandle, Era: e.Era, CreatedAt: e.CreatedAt,
	}
}

func toEvents(vs []usecase.EventView) []eventJSON {
	out := make([]eventJSON, len(vs))
	for i, v := range vs {
		out[i] = toEvent(v)
	}
	return out
}

type eventPageJSON struct {
	Items      []eventJSON `json:"items"`
	NextCursor *string     `json:"nextCursor"`
}

func toEventPage(p usecase.EventPage) eventPageJSON {
	return eventPageJSON{Items: toEvents(p.Items), NextCursor: p.NextCursor}
}

type bucketJSON struct {
	From   int         `json:"from"`
	To     int         `json:"to"`
	Count  int         `json:"count"`
	Sample []eventJSON `json:"sample"`
}

type eventWindowJSON struct {
	From    int                `json:"from"`
	To      int                `json:"to"`
	Total   int                `json:"total"`
	Mode    usecase.WindowMode `json:"mode"`
	Events  []eventJSON        `json:"events"`
	Buckets []bucketJSON       `json:"buckets"`
}

func toEventWindow(w usecase.EventWindow) eventWindowJSON {
	out := eventWindowJSON{From: w.From, To: w.To, Total: w.Total, Mode: w.Mode, Events: toEvents(w.Events), Buckets: make([]bucketJSON, len(w.Buckets))}
	for i, b := range w.Buckets {
		out.Buckets[i] = bucketJSON{From: b.From, To: b.To, Count: b.Count, Sample: toEvents(b.Sample)}
	}
	return out
}

type suggestionTimelineJSON struct {
	Slug  string            `json:"slug"`
	Title string            `json:"title"`
	Color domain.TrackColor `json:"color"`
}

type suggestionJSON struct {
	ID         string                  `json:"id"`
	Timeline   suggestionTimelineJSON  `json:"timeline"`
	Kind       domain.SuggestionKind   `json:"kind"`
	By         string                  `json:"by"`
	Status     domain.SuggestionStatus `json:"status"`
	EventID    *string                 `json:"eventId"`
	Event      eventInputJSON          `json:"event"`
	Note       *string                 `json:"note"`
	Feedback   *string                 `json:"feedback"`
	Applied    *string                 `json:"applied"`
	Previous   *eventJSON              `json:"previous"`
	CreatedAt  time.Time               `json:"createdAt"`
	ReviewedAt *time.Time              `json:"reviewedAt"`
}

func toSuggestion(v usecase.SuggestionView) suggestionJSON {
	s := v.Suggestion
	out := suggestionJSON{
		ID: s.ID, Timeline: suggestionTimelineJSON{v.Timeline.Slug, v.Timeline.Title, v.Timeline.Color},
		Kind: s.Kind, By: v.AuthorHandle, Status: s.Status, EventID: s.EventID, Event: toEventInput(s.Proposed),
		Note: s.Note, Feedback: s.Feedback, Applied: s.AppliedEventID, CreatedAt: s.CreatedAt, ReviewedAt: s.ReviewedAt,
	}
	if v.Previous != nil {
		p := toEvent(*v.Previous)
		out.Previous = &p
	}
	return out
}

func toSuggestions(vs []usecase.SuggestionView) []suggestionJSON {
	out := make([]suggestionJSON, len(vs))
	for i, v := range vs {
		out[i] = toSuggestion(v)
	}
	return out
}

type laneJSON struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Color    domain.TrackColor `json:"color"`
	Timeline timelineJSON      `json:"timeline"`
	eventWindowJSON
}

type overlapJSON struct {
	Year    int      `json:"year"`
	LaneIDs []string `json:"laneIds"`
}

type comparisonJSON struct {
	Start         int           `json:"start"`
	End           int           `json:"end"`
	Lanes         []laneJSON    `json:"lanes"`
	Overlaps      []overlapJSON `json:"overlaps"`
	OverlapsTotal int           `json:"overlapsTotal"`
}

func toComparison(v usecase.ComparisonView) comparisonJSON {
	out := comparisonJSON{Start: v.Start, End: v.End, Lanes: make([]laneJSON, len(v.Lanes)), Overlaps: make([]overlapJSON, len(v.Overlaps)), OverlapsTotal: v.OverlapsTotal}
	for i, l := range v.Lanes {
		out.Lanes[i] = laneJSON{ID: l.Timeline.Timeline.Slug, Name: l.Timeline.Timeline.Title, Color: l.Color, Timeline: toTimeline(l.Timeline), eventWindowJSON: toEventWindow(l.Window)}
	}
	for i, o := range v.Overlaps {
		out.Overlaps[i] = overlapJSON{Year: o.Year, LaneIDs: o.Slugs}
	}
	return out
}

type savedComparisonJSON struct {
	ID        string    `json:"id"`
	Timelines []string  `json:"timelines"`
	Start     int       `json:"start"`
	End       int       `json:"end"`
	CreatedAt time.Time `json:"createdAt"`
}

func toSavedComparison(c domain.SavedComparison) savedComparisonJSON {
	return savedComparisonJSON{ID: c.ID, Timelines: c.Slugs, Start: c.Start, End: c.End, CreatedAt: c.CreatedAt}
}

type tagJSON struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

type followingJSON struct {
	Users     []userSummaryJSON `json:"users"`
	Timelines []timelineJSON    `json:"timelines"`
}

type itemsJSON[T any] struct {
	Items []T `json:"items"`
}
