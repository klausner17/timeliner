package usecase

import (
	"context"
	"strings"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

type ActorInput struct{ ActorID string }

// GetMe devolve o usuário logado.
type GetMe struct{ d *Deps }

func NewGetMe(d *Deps) *GetMe { return &GetMe{d} }

func (uc *GetMe) Execute(ctx context.Context, in ActorInput) (domain.User, error) {
	if err := requireActor(in.ActorID); err != nil {
		return domain.User{}, err
	}
	u, err := uc.d.Users.ByID(ctx, in.ActorID)
	if err != nil {
		return domain.User{}, notFound(err, "Usuário não encontrado.")
	}
	return *u, nil
}

type UpdateProfileInput struct {
	ActorID string
	Name    *string
	Bio     *string
	Tint    *domain.Tint
}

type UpdateProfile struct{ d *Deps }

func NewUpdateProfile(d *Deps) *UpdateProfile { return &UpdateProfile{d} }

func (uc *UpdateProfile) Execute(ctx context.Context, in UpdateProfileInput) (domain.User, error) {
	if err := requireActor(in.ActorID); err != nil {
		return domain.User{}, err
	}
	u, err := uc.d.Users.ByID(ctx, in.ActorID)
	if err != nil {
		return domain.User{}, notFound(err, "Usuário não encontrado.")
	}
	f := fieldErrors{}
	if in.Name != nil {
		u.Name = strings.TrimSpace(*in.Name)
		f.check(length(u.Name, 1, 80), "name", "O nome precisa ter de 1 a 80 caracteres.")
	}
	if in.Bio != nil {
		u.Bio = strings.TrimSpace(*in.Bio)
		f.check(length(u.Bio, 0, 500), "bio", "A bio pode ter até 500 caracteres.")
	}
	if in.Tint != nil {
		u.Tint = *in.Tint
		f.check(u.Tint.Valid(), "tint", "Cor inválida.")
	}
	if err := f.err(); err != nil {
		return domain.User{}, err
	}
	if err := uc.d.Users.Update(ctx, u); err != nil {
		return domain.User{}, err
	}
	return *u, nil
}

type HandleInput struct {
	ActorID string
	Handle  string
}

// GetProfile é o perfil público com os números.
type GetProfile struct{ d *Deps }

func NewGetProfile(d *Deps) *GetProfile { return &GetProfile{d} }

func (uc *GetProfile) Execute(ctx context.Context, in HandleInput) (UserView, error) {
	u, err := uc.d.Users.ByHandle(ctx, in.Handle)
	if err != nil {
		return UserView{}, notFound(err, "Usuário não encontrado.")
	}
	stats, err := uc.d.Users.Stats(ctx, u.ID, in.ActorID)
	if err != nil {
		return UserView{}, err
	}
	return UserView{User: *u, Stats: stats}, nil
}

// ListContributions são as sugestões que o usuário enviou, nas timelines que o viewer vê.
type ListContributions struct{ d *Deps }

func NewListContributions(d *Deps) *ListContributions { return &ListContributions{d} }

func (uc *ListContributions) Execute(ctx context.Context, in HandleInput) ([]SuggestionView, error) {
	u, err := uc.d.Users.ByHandle(ctx, in.Handle)
	if err != nil {
		return nil, notFound(err, "Usuário não encontrado.")
	}
	ss, err := uc.d.Suggestions.ListByAuthor(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return suggestionViews(ctx, uc.d, ss, in.ActorID)
}
