package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

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

type FavoriteTimelineInput struct {
	ActorID string
	Slug    string
	// Favorite false tira a estrela.
	Favorite bool
}

// FavoriteTimeline dá ou tira a estrela. É como ler: vale para quem ainda não confirmou o
// e-mail e para as próprias timelines.
type FavoriteTimeline struct{ d *Deps }

func NewFavoriteTimeline(d *Deps) *FavoriteTimeline { return &FavoriteTimeline{d} }

func (uc *FavoriteTimeline) Execute(ctx context.Context, in FavoriteTimelineInput) (Empty, error) {
	if err := requireActor(in.ActorID); err != nil {
		return Empty{}, err
	}
	t, err := visibleTimeline(ctx, uc.d, in.Slug, in.ActorID)
	if err != nil {
		return Empty{}, err
	}
	if in.Favorite {
		return Empty{}, uc.d.Follows.FavoriteTimeline(ctx, in.ActorID, t.ID)
	}
	return Empty{}, uc.d.Follows.UnfavoriteTimeline(ctx, in.ActorID, t.ID)
}

type RecordViewInput struct {
	ActorID string
	// ClientIP identifica o visitante sem login. Não é guardado: vira um hash com o dia.
	ClientIP string
	Slug     string
}

// RecordView conta que alguém abriu a timeline: uma vez por pessoa por dia. O dono
// vendo a própria não conta.
type RecordView struct{ d *Deps }

func NewRecordView(d *Deps) *RecordView { return &RecordView{d} }

func (uc *RecordView) Execute(ctx context.Context, in RecordViewInput) (Empty, error) {
	t, err := visibleTimeline(ctx, uc.d, in.Slug, in.ActorID)
	if err != nil {
		return Empty{}, err
	}
	if in.ActorID != "" && in.ActorID == t.OwnerID {
		return Empty{}, nil
	}
	day := uc.d.Clock.Now().UTC().Format(time.DateOnly)
	viewer := "u:" + in.ActorID
	if in.ActorID == "" {
		if in.ClientIP == "" {
			return Empty{}, nil
		}
		sum := sha256.Sum256([]byte(day + "|" + in.ClientIP))
		viewer = "ip:" + hex.EncodeToString(sum[:12])
	}
	return Empty{}, uc.d.Timelines.RecordView(ctx, t.ID, viewer, day)
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
	Users          []domain.User
	Timelines      []TimelineView
	UsersTotal     int
	TimelinesTotal int
	Page           int
	PageSize       int
}

// ListFollowing é o que o usuário segue. Timelines que o viewer não vê ficam de fora.
type ListFollowing struct{ d *Deps }

func NewListFollowing(d *Deps) *ListFollowing { return &ListFollowing{d} }

type ListFollowingInput struct {
	ActorID string
	Handle  string
	// A mesma página vale para as duas listas (pessoas e timelines).
	PageInput
}

func (uc *ListFollowing) Execute(ctx context.Context, in ListFollowingInput) (FollowingView, error) {
	f := fieldErrors{}
	in.normalize(f, 50, 100)
	if err := f.err(); err != nil {
		return FollowingView{}, err
	}
	u, err := uc.d.Users.ByHandle(ctx, in.Handle)
	if err != nil {
		return FollowingView{}, notFound(err, "Usuário não encontrado.")
	}
	userIDs, usersTotal, err := uc.d.Follows.FollowedUsers(ctx, u.ID, in.paging())
	if err != nil {
		return FollowingView{}, err
	}
	users, err := uc.d.Users.ByIDs(ctx, userIDs)
	if err != nil {
		return FollowingView{}, err
	}
	out := FollowingView{Users: []domain.User{}, Timelines: []TimelineView{}, UsersTotal: usersTotal, Page: in.Page, PageSize: in.PageSize}
	for _, id := range userIDs {
		if fu, ok := users[id]; ok {
			out.Users = append(out.Users, fu)
		}
	}
	p := in.paging()
	ts, timelinesTotal, err := uc.d.Timelines.List(ctx, TimelineFilter{FollowedBy: u.ID, ViewerID: in.ActorID, Limit: p.Limit, Offset: p.Offset})
	if err != nil {
		return FollowingView{}, err
	}
	out.TimelinesTotal = timelinesTotal
	if out.Timelines, err = timelineViews(ctx, uc.d, ts, in.ActorID); err != nil {
		return FollowingView{}, err
	}
	return out, nil
}
