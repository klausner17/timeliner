package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time {
	c.t = c.t.Add(time.Second)
	return c.t
}

type seqIDs struct{ n int }

func (s *seqIDs) NewID(p string) string { s.n++; return fmt.Sprintf("%s_%d", p, s.n) }
func (s *seqIDs) NewToken() string      { s.n++; return fmt.Sprintf("token%d", s.n) }

type plainHasher struct{}

func (plainHasher) Hash(p string) (string, error) { return "h:" + p, nil }
func (plainHasher) Compare(h, p string) bool      { return h == "h:"+p }

func newDeps() *usecase.Deps {
	d := repositories()
	d.Clock = &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	d.IDs = &seqIDs{}
	d.Hasher = plainHasher{}
	return d
}

var ctx = context.Background()

func ptr[T any](v T) *T { return &v }

func must[O any](t *testing.T) func(O, error) O {
	t.Helper()
	return func(o O, err error) O {
		t.Helper()
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		return o
	}
}

func wantKind(t *testing.T, err error, kind domain.Kind) {
	t.Helper()
	if got := domain.KindOf(err); got != kind {
		t.Fatalf("erro = %v (kind %q), queria kind %q", err, got, kind)
	}
}

func register(t *testing.T, d *usecase.Deps, handle string) domain.User {
	t.Helper()
	out := must[usecase.AuthOutput](t)(usecase.NewRegister(d).Execute(ctx, usecase.RegisterInput{
		Handle: handle, Name: handle, Email: handle + "@exemplo.com", Password: "senha-forte",
	}))
	return out.User
}

func createTimeline(t *testing.T, d *usecase.Deps, owner, title string, events ...domain.EventData) usecase.TimelineDetailView {
	t.Helper()
	return must[usecase.TimelineDetailView](t)(usecase.NewCreateTimeline(d).Execute(ctx, usecase.CreateTimelineInput{
		ActorID:        owner,
		TimelineFields: usecase.TimelineFields{Title: &title, Category: ptr("História")},
		Events:         events,
	}))
}

func ev(year int, title string) domain.EventData { return domain.EventData{Year: year, Title: title} }
