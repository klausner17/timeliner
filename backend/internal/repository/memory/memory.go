// Package memory implementa os repositories em memória. Serve para testes de
// usecase e de controller; não tem rollback de transação.
package memory

import (
	"cmp"
	"context"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

type db struct {
	mu          sync.RWMutex
	users       map[string]domain.User
	sessions    map[string]domain.Session
	timelines   map[string]domain.Timeline
	events      map[string]domain.Event
	suggestions map[string]domain.Suggestion
	// follows de timeline e de usuário: chave seguidor+"|"+seguido, valor a ordem.
	timelineFollows map[string]int
	userFollows     map[string]int
	comparisons     map[string]domain.SavedComparison
	calendars       map[string]domain.Calendar
	seq             int
}

// New devolve os repositories ligados a um mesmo banco em memória.
func New() *usecase.Deps {
	d := &db{
		users: map[string]domain.User{}, sessions: map[string]domain.Session{},
		timelines: map[string]domain.Timeline{}, events: map[string]domain.Event{},
		suggestions: map[string]domain.Suggestion{}, timelineFollows: map[string]int{},
		userFollows: map[string]int{}, comparisons: map[string]domain.SavedComparison{},
		calendars: map[string]domain.Calendar{},
	}
	return &usecase.Deps{
		Users: users{d}, Sessions: sessions{d}, Timelines: timelines{d}, Calendars: calendars{d}, Events: events{d},
		Suggestions: suggestions{d}, Follows: follows{d}, Comparisons: comparisons{d}, Tx: tx{},
	}
}

type tx struct{}

func (tx) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

// Users

type users struct{ *db }

func (r users) Create(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.users {
		if o.Handle == u.Handle || o.Email == u.Email {
			return domain.ErrDuplicate
		}
	}
	r.users[u.ID] = *u
	return nil
}

func (r users) Update(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[u.ID]; !ok {
		return domain.ErrNotFound
	}
	r.users[u.ID] = *u
	return nil
}

func (r users) ByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return found(r.users[id], r.users[id].ID != "")
}

func (r users) find(match func(domain.User) bool) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if match(u) {
			return &u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r users) ByHandle(_ context.Context, h string) (*domain.User, error) {
	return r.find(func(u domain.User) bool { return u.Handle == h })
}

func (r users) ByEmail(_ context.Context, e string) (*domain.User, error) {
	return r.find(func(u domain.User) bool { return u.Email == e })
}

func (r users) ByIDs(_ context.Context, ids []string) (map[string]domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := map[string]domain.User{}
	for _, id := range ids {
		if u, ok := r.users[id]; ok {
			out[id] = u
		}
	}
	return out, nil
}

func (r users) Stats(_ context.Context, userID, viewerID string) (domain.UserStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var s domain.UserStats
	for _, t := range r.timelines {
		if t.OwnerID == userID && t.Listed() {
			s.Timelines++
		}
	}
	for _, e := range r.events {
		if e.AuthorID == userID {
			s.Events++
		}
	}
	for _, sg := range r.suggestions {
		if sg.AuthorID == userID && sg.Status == domain.StatusApproved {
			s.ApprovedContributions++
		}
	}
	for k := range r.userFollows {
		if strings.HasSuffix(k, "|"+userID) {
			s.Followers++
		}
	}
	_, s.Following = r.userFollows[viewerID+"|"+userID]
	return s, nil
}

// Sessions

type sessions struct{ *db }

func (r sessions) Create(_ context.Context, s domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[s.TokenHash] = s
	return nil
}

func (r sessions) UserID(_ context.Context, hash string, now time.Time) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sessions[hash]
	if !ok || !now.Before(s.ExpiresAt) {
		return "", domain.ErrNotFound
	}
	return s.UserID, nil
}

// Timelines

type timelines struct{ *db }

func (r timelines) Create(_ context.Context, t *domain.Timeline) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.timelines {
		if o.Slug == t.Slug {
			return domain.ErrDuplicate
		}
	}
	r.timelines[t.ID] = clone(*t)
	return nil
}

func (r timelines) Update(_ context.Context, t *domain.Timeline) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.timelines[t.ID]; !ok {
		return domain.ErrNotFound
	}
	r.timelines[t.ID] = clone(*t)
	return nil
}

func (r timelines) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.timelines[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.timelines, id)
	for k, e := range r.events {
		if e.TimelineID == id {
			delete(r.events, k)
		}
	}
	for k, s := range r.suggestions {
		if s.TimelineID == id {
			delete(r.suggestions, k)
		}
	}
	for k := range r.timelineFollows {
		if strings.HasSuffix(k, "|"+id) {
			delete(r.timelineFollows, k)
		}
	}
	return nil
}

func (r timelines) ByID(_ context.Context, id string) (*domain.Timeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.timelines[id]
	return found(clone(t), ok)
}

func (r timelines) BySlug(_ context.Context, slug string) (*domain.Timeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, t := range r.timelines {
		if t.Slug == slug {
			return found(clone(t), true)
		}
	}
	return nil, domain.ErrNotFound
}

func (r timelines) SlugTaken(ctx context.Context, slug string) (bool, error) {
	_, err := r.BySlug(ctx, slug)
	return err == nil, nil
}

func (r timelines) List(_ context.Context, f usecase.TimelineFilter) ([]domain.Timeline, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	q := strings.ToLower(f.Query)
	matched := []domain.Timeline{}
	for _, t := range r.timelines {
		switch {
		case !f.IncludeUnlisted && !t.Listed(),
			f.OwnerID != "" && t.OwnerID != f.OwnerID,
			f.Category != "" && t.Category != f.Category,
			f.Tag != "" && !slices.Contains(t.Tags, f.Tag),
			f.OnlyOpen && t.Contributions != domain.ContributionsSuggestions,
			q != "" && !strings.Contains(strings.ToLower(t.Title+" "+t.Description+" "+strings.Join(t.Tags, " ")), q):
			continue
		}
		matched = append(matched, clone(t))
	}
	followers := map[string]int{}
	for k := range r.timelineFollows {
		followers[k[strings.Index(k, "|")+1:]]++
	}
	slices.SortFunc(matched, func(a, b domain.Timeline) int {
		if f.Sort == usecase.SortPopular {
			if c := cmp.Compare(followers[b.ID], followers[a.ID]); c != 0 {
				return c
			}
		}
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	total := len(matched)
	lo := min(f.Offset, total)
	hi := min(lo+f.Limit, total)
	return matched[lo:hi], total, nil
}

func (r timelines) Stats(_ context.Context, ids []string, viewerID string) (map[string]domain.TimelineStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := map[string]domain.TimelineStats{}
	for _, id := range ids {
		t := r.timelines[id]
		s := domain.TimelineStats{Positions: []int{}}
		years := []int{}
		authors := map[string]bool{}
		for _, e := range r.events {
			if e.TimelineID != id {
				continue
			}
			s.EventCount++
			years = append(years, e.Year)
			if e.AuthorID != t.OwnerID {
				authors[e.AuthorID] = true
			}
			end := e.Year
			if e.EndYear != nil {
				end = *e.EndYear
			}
			if s.StartYear == nil || e.Year < *s.StartYear {
				s.StartYear = &e.Year
			}
			if s.EndYear == nil || end > *s.EndYear {
				s.EndYear = &end
			}
		}
		for _, y := range years {
			p := 500
			if *s.EndYear > *s.StartYear {
				p = int(math.Round(float64(y-*s.StartYear) * 1000 / float64(*s.EndYear-*s.StartYear)))
			}
			if !slices.Contains(s.Positions, p) {
				s.Positions = append(s.Positions, p)
			}
		}
		slices.Sort(s.Positions)
		s.ContributorCount = len(authors)
		for _, sg := range r.suggestions {
			if sg.TimelineID == id && sg.Status == domain.StatusPending {
				s.PendingSuggestions++
			}
		}
		for k := range r.timelineFollows {
			if strings.HasSuffix(k, "|"+id) {
				s.Followers++
			}
		}
		_, s.Following = r.timelineFollows[viewerID+"|"+id]
		out[id] = s
	}
	return out, nil
}

func (r timelines) Tags(_ context.Context, limit int) ([]domain.TagCount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	counts := map[string]int{}
	for _, t := range r.timelines {
		if t.Listed() {
			for _, tag := range t.Tags {
				counts[tag]++
			}
		}
	}
	out := []domain.TagCount{}
	for tag, n := range counts {
		out = append(out, domain.TagCount{Tag: tag, Count: n})
	}
	slices.SortFunc(out, func(a, b domain.TagCount) int {
		if c := cmp.Compare(b.Count, a.Count); c != 0 {
			return c
		}
		return cmp.Compare(a.Tag, b.Tag)
	})
	return out[:min(limit, len(out))], nil
}

// Events

type events struct{ *db }

func (r events) Create(_ context.Context, e *domain.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events[e.ID] = *e
	return nil
}

func (r events) Update(_ context.Context, e *domain.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.events[e.ID]; !ok {
		return domain.ErrNotFound
	}
	r.events[e.ID] = *e
	return nil
}

func (r events) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.events[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.events, id)
	return nil
}

func (r events) ByID(_ context.Context, id string) (*domain.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.events[id]
	return found(e, ok)
}

func (r events) RemapEras(_ context.Context, timelineID string, remap map[string]*string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, e := range r.events {
		if e.TimelineID != timelineID || e.Era == nil {
			continue
		}
		if to, ok := remap[*e.Era]; ok {
			e.Era = to
			r.events[id] = e
		}
	}
	return nil
}

func (r events) CreateMany(ctx context.Context, es []domain.Event) error {
	for i := range es {
		if err := r.Create(ctx, &es[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r events) LatestYearIn(_ context.Context, timelineIDs []string, from, to int) (*int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var latest *int
	for _, e := range r.events {
		if !slices.Contains(timelineIDs, e.TimelineID) {
			continue
		}
		for _, y := range []int{e.Year, e.End()} {
			if y >= from && y <= to && (latest == nil || y > *latest) {
				latest = &y
			}
		}
	}
	return latest, nil
}

func (r events) ShiftYears(_ context.Context, timelineIDs []string, shifts []domain.YearShift) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, e := range r.events {
		if slices.Contains(timelineIDs, e.TimelineID) {
			e.EventData = shiftData(e.EventData, shifts)
			r.events[id] = e
		}
	}
	return nil
}

// touching devolve, em ordem cronológica, os eventos da timeline que tocam [from, to].
func (r events) touching(timelineID string, from, to *int) []domain.Event {
	out := []domain.Event{}
	for _, e := range r.events {
		if e.TimelineID != timelineID || (from != nil && e.End() < *from) || (to != nil && e.Year > *to) {
			continue
		}
		out = append(out, e)
	}
	SortEvents(out)
	return out
}

func (r events) ListByTimeline(_ context.Context, q usecase.EventQuery) ([]domain.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := r.touching(q.TimelineID, q.From, q.To)
	if q.After != nil {
		out = out[sort.Search(len(out), func(i int) bool { return compareCursor(domain.CursorOf(out[i]), *q.After) > 0 }):]
	}
	return out[:min(q.Limit, len(out))], nil
}

func (r events) Summary(_ context.Context, q usecase.SummaryQuery) ([]domain.YearBucket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	span := q.To - q.From + 1
	out := []domain.YearBucket{}
	last := -1
	for _, e := range r.touching(q.TimelineID, &q.From, &q.To) {
		if b := domain.BucketOf(e.Year, q.From, span, q.Buckets); b != last {
			from, to := domain.BucketRange(b, q.From, span, q.Buckets)
			out = append(out, domain.YearBucket{From: from, To: to, Sample: []domain.Event{}})
			last = b
		}
		cur := &out[len(out)-1]
		cur.Count++
		if len(cur.Sample) < q.Sample {
			cur.Sample = append(cur.Sample, e)
		}
	}
	return out, nil
}

func (r events) Overlaps(_ context.Context, timelineIDs []string, from, to, limit int) ([]domain.YearOverlap, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	byYear := map[int][]string{}
	for _, e := range r.events {
		if slices.Contains(timelineIDs, e.TimelineID) && e.Year >= from && e.Year <= to && !slices.Contains(byYear[e.Year], e.TimelineID) {
			byYear[e.Year] = append(byYear[e.Year], e.TimelineID)
		}
	}
	out := []domain.YearOverlap{}
	for year, ids := range byYear {
		if len(ids) >= 2 {
			slices.Sort(ids)
			out = append(out, domain.YearOverlap{Year: year, TimelineIDs: ids})
		}
	}
	slices.SortFunc(out, func(a, b domain.YearOverlap) int { return a.Year - b.Year })
	return out[:min(limit, len(out))], len(out), nil
}

// SortEvents é a ordem cronológica usada pelos dois repositories: ano, mês, dia, hora,
// fim, criação, ID.
func SortEvents(es []domain.Event) {
	slices.SortFunc(es, func(a, b domain.Event) int { return compareCursor(domain.CursorOf(a), domain.CursorOf(b)) })
}

func compareCursor(a, b domain.EventCursor) int {
	return cmp.Or(cmp.Compare(a.Year, b.Year), cmp.Compare(a.Month, b.Month), cmp.Compare(a.Day, b.Day), cmp.Compare(a.Time, b.Time),
		cmp.Compare(a.EndYear, b.EndYear), a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.ID, b.ID))
}

func (r timelines) ByCalendar(_ context.Context, calendarID string) ([]domain.Timeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []domain.Timeline{}
	for _, t := range r.timelines {
		if t.CalendarID != nil && *t.CalendarID == calendarID {
			out = append(out, clone(t))
		}
	}
	slices.SortFunc(out, func(a, b domain.Timeline) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

// Calendars

type calendars struct{ *db }

func cloneCalendar(c domain.Calendar) domain.Calendar {
	c.Eras = slices.Clone(c.Eras)
	return c
}

func (r calendars) Create(_ context.Context, c *domain.Calendar) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calendars[c.ID] = cloneCalendar(*c)
	return nil
}

func (r calendars) Update(_ context.Context, c *domain.Calendar) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.calendars[c.ID]; !ok {
		return domain.ErrNotFound
	}
	r.calendars[c.ID] = cloneCalendar(*c)
	return nil
}

func (r calendars) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.calendars[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.calendars, id)
	return nil
}

func (r calendars) ByID(_ context.Context, id string) (*domain.Calendar, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.calendars[id]
	return found(cloneCalendar(c), ok)
}

func (r calendars) ByIDs(_ context.Context, ids []string) (map[string]domain.Calendar, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := map[string]domain.Calendar{}
	for _, id := range ids {
		if c, ok := r.calendars[id]; ok {
			out[id] = cloneCalendar(c)
		}
	}
	return out, nil
}

func (r calendars) Usable(_ context.Context, userID, query string) ([]domain.Calendar, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	q := strings.ToLower(query)
	out := []domain.Calendar{}
	for _, c := range r.calendars {
		if (c.Shared || (userID != "" && c.OwnerID == userID)) && strings.Contains(strings.ToLower(c.Name), q) {
			out = append(out, cloneCalendar(c))
		}
	}
	slices.SortFunc(out, func(a, b domain.Calendar) int {
		return cmp.Or(cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)), cmp.Compare(a.ID, b.ID))
	})
	return out, nil
}

func (r calendars) InUse(_ context.Context, id string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, t := range r.timelines {
		if t.CalendarID != nil && *t.CalendarID == id {
			return true, nil
		}
	}
	return false, nil
}

// Suggestions

type suggestions struct{ *db }

func (r suggestions) ShiftYears(_ context.Context, timelineIDs []string, shifts []domain.YearShift) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, s := range r.suggestions {
		if !slices.Contains(timelineIDs, s.TimelineID) {
			continue
		}
		s.Proposed = shiftData(s.Proposed, shifts)
		if s.Previous != nil {
			p := *s.Previous
			p.EventData = shiftData(p.EventData, shifts)
			s.Previous = &p
		}
		r.suggestions[id] = s
	}
	return nil
}

func shiftData(e domain.EventData, shifts []domain.YearShift) domain.EventData {
	e.Year = domain.ShiftYear(e.Year, shifts)
	if e.EndYear != nil {
		end := domain.ShiftYear(*e.EndYear, shifts)
		e.EndYear = &end
	}
	return e
}

func (r suggestions) Create(_ context.Context, s *domain.Suggestion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.suggestions[s.ID] = *s
	return nil
}

func (r suggestions) Update(_ context.Context, s *domain.Suggestion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.suggestions[s.ID]; !ok {
		return domain.ErrNotFound
	}
	r.suggestions[s.ID] = *s
	return nil
}

func (r suggestions) ByID(_ context.Context, id string) (*domain.Suggestion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.suggestions[id]
	return found(s, ok)
}

func (r suggestions) list(match func(domain.Suggestion) bool) []domain.Suggestion {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []domain.Suggestion{}
	for _, s := range r.suggestions {
		if match(s) {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b domain.Suggestion) int {
		return cmp.Or(b.CreatedAt.Compare(a.CreatedAt), cmp.Compare(b.ID, a.ID))
	})
	return out
}

func (r suggestions) ListByTimeline(_ context.Context, timelineID string, status domain.SuggestionStatus) ([]domain.Suggestion, error) {
	return r.list(func(s domain.Suggestion) bool {
		return s.TimelineID == timelineID && (status == "" || s.Status == status)
	}), nil
}

func (r suggestions) ListByAuthor(_ context.Context, authorID string) ([]domain.Suggestion, error) {
	return r.list(func(s domain.Suggestion) bool { return s.AuthorID == authorID }), nil
}

// Follows

type follows struct{ *db }

func (r follows) add(m map[string]int, a, b string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := m[a+"|"+b]; !ok {
		r.seq++
		m[a+"|"+b] = r.seq
	}
	return nil
}

func (r follows) remove(m map[string]int, a, b string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(m, a+"|"+b)
	return nil
}

func (r follows) followed(m map[string]int, a string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	type item struct {
		id  string
		seq int
	}
	items := []item{}
	for k, seq := range m {
		if strings.HasPrefix(k, a+"|") {
			items = append(items, item{k[len(a)+1:], seq})
		}
	}
	slices.SortFunc(items, func(x, y item) int { return cmp.Compare(y.seq, x.seq) })
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.id
	}
	return out
}

func (r follows) FollowTimeline(_ context.Context, u, t string) error {
	return r.add(r.timelineFollows, u, t)
}
func (r follows) UnfollowTimeline(_ context.Context, u, t string) error {
	return r.remove(r.timelineFollows, u, t)
}
func (r follows) FollowUser(_ context.Context, a, b string) error { return r.add(r.userFollows, a, b) }
func (r follows) UnfollowUser(_ context.Context, a, b string) error {
	return r.remove(r.userFollows, a, b)
}
func (r follows) FollowedTimelines(_ context.Context, u string) ([]string, error) {
	return r.followed(r.timelineFollows, u), nil
}
func (r follows) FollowedUsers(_ context.Context, u string) ([]string, error) {
	return r.followed(r.userFollows, u), nil
}

// Comparisons

type comparisons struct{ *db }

func (r comparisons) Create(_ context.Context, c *domain.SavedComparison) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.comparisons[c.ID] = *c
	return nil
}

func (r comparisons) ListByUser(_ context.Context, userID string) ([]domain.SavedComparison, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []domain.SavedComparison{}
	for _, c := range r.comparisons {
		if c.UserID == userID {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b domain.SavedComparison) int {
		return cmp.Or(b.CreatedAt.Compare(a.CreatedAt), cmp.Compare(b.ID, a.ID))
	})
	return out, nil
}

func (r comparisons) Delete(_ context.Context, id, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.comparisons[id]
	if !ok || c.UserID != userID {
		return domain.ErrNotFound
	}
	delete(r.comparisons, id)
	return nil
}

func found[T any](v T, ok bool) (*T, error) {
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &v, nil
}

// clone copia as fatias para que quem chama não altere o que está guardado.
func clone(t domain.Timeline) domain.Timeline {
	t.Eras = slices.Clone(t.Eras)
	t.Tags = slices.Clone(t.Tags)
	return t
}
