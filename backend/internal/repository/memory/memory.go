// Package memory implementa os repositories em memória. Serve para testes de
// usecase e de controller; não tem rollback de transação.
package memory

import (
	"cmp"
	"context"
	"slices"
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
	seq             int
}

// New devolve os repositories ligados a um mesmo banco em memória.
func New() *usecase.Deps {
	d := &db{
		users: map[string]domain.User{}, sessions: map[string]domain.Session{},
		timelines: map[string]domain.Timeline{}, events: map[string]domain.Event{},
		suggestions: map[string]domain.Suggestion{}, timelineFollows: map[string]int{},
		userFollows: map[string]int{}, comparisons: map[string]domain.SavedComparison{},
	}
	return &usecase.Deps{
		Users: users{d}, Sessions: sessions{d}, Timelines: timelines{d}, Events: events{d},
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
		s := domain.TimelineStats{Years: []int{}}
		authors := map[string]bool{}
		for _, e := range r.events {
			if e.TimelineID != id {
				continue
			}
			s.EventCount++
			s.Years = append(s.Years, e.Year)
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
		slices.Sort(s.Years)
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

func (r events) ListByTimeline(_ context.Context, timelineID string, from, to *int) ([]domain.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []domain.Event{}
	for _, e := range r.events {
		end := e.Year
		if e.EndYear != nil {
			end = *e.EndYear
		}
		if e.TimelineID != timelineID || (from != nil && end < *from) || (to != nil && e.Year > *to) {
			continue
		}
		out = append(out, e)
	}
	SortEvents(out)
	return out, nil
}

// SortEvents é a ordem cronológica usada pelos dois repositories: ano, fim, criação, ID.
func SortEvents(es []domain.Event) {
	slices.SortFunc(es, func(a, b domain.Event) int {
		return cmp.Or(
			cmp.Compare(a.Year, b.Year),
			cmp.Compare(endOf(a), endOf(b)),
			a.CreatedAt.Compare(b.CreatedAt),
			cmp.Compare(a.ID, b.ID),
		)
	})
}

func endOf(e domain.Event) int {
	if e.EndYear != nil {
		return *e.EndYear
	}
	return e.Year
}

// Suggestions

type suggestions struct{ *db }

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
