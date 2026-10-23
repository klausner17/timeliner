package usecase

import (
	"context"
	"errors"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

type FollowTimelineInput struct {
	ActorID string
	Slug    string
	// Follow false deixa de seguir.
	Follow bool
}

type FollowTimeline struct{ d *Deps }

func NewFollowTimeline(d *Deps) *FollowTimeline { return &FollowTimeline{d} }

func (uc *FollowTimeline) Execute(ctx context.Context, in FollowTimelineInput) (Empty, error) {
	if err := requireActor(in.ActorID); err != nil {
		return Empty{}, err
	}
	t, err := visibleTimeline(ctx, uc.d, in.Slug, in.ActorID)
	if err != nil {
		return Empty{}, err
	}
	if in.Follow {
		return Empty{}, uc.d.Follows.FollowTimeline(ctx, in.ActorID, t.ID)
	}
	return Empty{}, uc.d.Follows.UnfollowTimeline(ctx, in.ActorID, t.ID)
}

type FollowUserInput struct {
	ActorID string
	Handle  string
	Follow  bool
}

type FollowUser struct{ d *Deps }

func NewFollowUser(d *Deps) *FollowUser { return &FollowUser{d} }

func (uc *FollowUser) Execute(ctx context.Context, in FollowUserInput) (Empty, error) {
	if err := requireActor(in.ActorID); err != nil {
		return Empty{}, err
	}
	u, err := uc.d.Users.ByHandle(ctx, in.Handle)
	if err != nil {
		return Empty{}, notFound(err, "Usuário não encontrado.")
	}
	if !in.Follow {
		return Empty{}, uc.d.Follows.UnfollowUser(ctx, in.ActorID, u.ID)
	}
	if u.ID == in.ActorID {
		return Empty{}, domain.Invalid(map[string]string{"handle": "Você não pode seguir a si mesmo."})
	}
	return Empty{}, uc.d.Follows.FollowUser(ctx, in.ActorID, u.ID)
}

type FollowingView struct {
	Users     []domain.User
	Timelines []TimelineView
}

// ListFollowing é o que o usuário segue. Timelines que o viewer não vê ficam de fora.
type ListFollowing struct{ d *Deps }

func NewListFollowing(d *Deps) *ListFollowing { return &ListFollowing{d} }

func (uc *ListFollowing) Execute(ctx context.Context, in HandleInput) (FollowingView, error) {
	u, err := uc.d.Users.ByHandle(ctx, in.Handle)
	if err != nil {
		return FollowingView{}, notFound(err, "Usuário não encontrado.")
	}
	userIDs, err := uc.d.Follows.FollowedUsers(ctx, u.ID)
	if err != nil {
		return FollowingView{}, err
	}
	users, err := uc.d.Users.ByIDs(ctx, userIDs)
	if err != nil {
		return FollowingView{}, err
	}
	out := FollowingView{Users: []domain.User{}, Timelines: []TimelineView{}}
	for _, id := range userIDs {
		if fu, ok := users[id]; ok {
			out.Users = append(out.Users, fu)
		}
	}
	timelineIDs, err := uc.d.Follows.FollowedTimelines(ctx, u.ID)
	if err != nil {
		return FollowingView{}, err
	}
	ts := []domain.Timeline{}
	for _, id := range timelineIDs {
		t, err := uc.d.Timelines.ByID(ctx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			return FollowingView{}, err
		}
		if t.VisibleTo(in.ActorID) {
			ts = append(ts, *t)
		}
	}
	if out.Timelines, err = timelineViews(ctx, uc.d, ts, in.ActorID); err != nil {
		return FollowingView{}, err
	}
	return out, nil
}
