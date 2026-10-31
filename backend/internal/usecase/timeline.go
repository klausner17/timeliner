package usecase

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

type ListTimelinesInput struct {
	ActorID string
	Query   string
	// Category é o filtro "cat" do explorar.
	Category    string
	Tag         string
	OwnerHandle string
	OnlyOpen    bool
	Sort        TimelineSort
	Page        int
	PageSize    int
}

type TimelinePage struct {
	Items    []TimelineView
	Page     int
	PageSize int
	Total    int
}

// ListTimelines é o explorar. Quando o dono lista as próprias (owner = seu handle ou "me"),
// vêm também rascunhos, privadas e as de link.
type ListTimelines struct{ d *Deps }

func NewListTimelines(d *Deps) *ListTimelines { return &ListTimelines{d} }

func (uc *ListTimelines) Execute(ctx context.Context, in ListTimelinesInput) (TimelinePage, error) {
	f := fieldErrors{}
	if in.Page == 0 {
		in.Page = 1
	}
	if in.PageSize == 0 {
		in.PageSize = 20
	}
	if in.Sort == "" {
		in.Sort = SortPopular
	}
	f.check(in.Page >= 1, "page", "A página começa em 1.")
	f.check(in.PageSize >= 1 && in.PageSize <= 100, "pageSize", "Use de 1 a 100.")
	f.check(in.Sort == SortPopular || in.Sort == SortRecent, "sort", "Use popular ou recent.")
	f.check(in.Category == "" || domain.ValidCategory(in.Category), "cat", "Categoria inválida.")
	if err := f.err(); err != nil {
		return TimelinePage{}, err
	}
	filter := TimelineFilter{
		Query: strings.TrimSpace(in.Query), Category: in.Category, Tag: strings.ToLower(strings.TrimSpace(in.Tag)),
		OnlyOpen: in.OnlyOpen, Sort: in.Sort, Limit: in.PageSize, Offset: (in.Page - 1) * in.PageSize,
	}
	switch {
	case in.OwnerHandle == "me":
		if err := requireActor(in.ActorID); err != nil {
			return TimelinePage{}, err
		}
		filter.OwnerID, filter.IncludeUnlisted = in.ActorID, true
	case in.OwnerHandle != "":
		owner, err := uc.d.Users.ByHandle(ctx, in.OwnerHandle)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return TimelinePage{Items: []TimelineView{}, Page: in.Page, PageSize: in.PageSize}, nil
			}
			return TimelinePage{}, err
		}
		filter.OwnerID = owner.ID
		filter.IncludeUnlisted = owner.ID == in.ActorID
	}
	ts, total, err := uc.d.Timelines.List(ctx, filter)
	if err != nil {
		return TimelinePage{}, err
	}
	items, err := timelineViews(ctx, uc.d, ts, in.ActorID)
	if err != nil {
		return TimelinePage{}, err
	}
	return TimelinePage{Items: items, Page: in.Page, PageSize: in.PageSize, Total: total}, nil
}

type TimelineFields struct {
	Title         *string
	Description   *string
	Category      *string
	Color         *domain.TrackColor
	Visibility    *domain.Visibility
	Contributions *domain.ContributionMode
	Draft         *bool
	Eras          *[]domain.Era
	// RenamedEras liga o nome novo de um período ao nome que ele tinha antes, para os
	// eventos acompanharem a troca de nome. Só vale junto com Eras.
	RenamedEras map[string]string
	Tags        *[]string
}

// MaxEras é quantos períodos uma timeline pode ter.
const MaxEras = 60

// apply copia os campos preenchidos para t e valida o resultado.
func (in TimelineFields) apply(t *domain.Timeline) error {
	f := fieldErrors{}
	if in.Title != nil {
		t.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		t.Description = strings.TrimSpace(*in.Description)
	}
	if in.Category != nil {
		t.Category = *in.Category
	}
	if in.Color != nil {
		t.Color = *in.Color
	}
	if in.Visibility != nil {
		t.Visibility = *in.Visibility
	}
	if in.Contributions != nil {
		t.Contributions = *in.Contributions
	}
	if in.Draft != nil {
		t.Draft = *in.Draft
	}
	if in.Eras != nil {
		t.Eras = make([]domain.Era, len(*in.Eras))
		for i, era := range *in.Eras {
			era.Name = strings.TrimSpace(era.Name)
			t.Eras[i] = era
		}
	}
	if in.Tags != nil {
		t.Tags = normalizeTags(*in.Tags)
	}
	f.check(length(t.Title, 1, 120), "title", "O título precisa ter de 1 a 120 caracteres.")
	f.check(length(t.Description, 0, 2000), "description", "A descrição pode ter até 2000 caracteres.")
	f.check(domain.ValidCategory(t.Category), "category", "Categoria inválida.")
	f.check(t.Color.Valid(), "color", "Cor inválida.")
	f.check(t.Visibility.Valid(), "visibility", "Visibilidade inválida.")
	f.check(t.Contributions.Valid(), "contributions", "Modo de contribuição inválido.")
	f.check(len(t.Tags) <= 10, "tags", "No máximo 10 tags.")
	for _, tag := range t.Tags {
		f.check(length(tag, 1, 30), "tags", "Cada tag pode ter até 30 caracteres.")
	}
	f.check(len(t.Eras) <= MaxEras, "eras", fmt.Sprintf("No máximo %d períodos.", MaxEras))
	names := map[string]bool{}
	for i, era := range t.Eras {
		p := fmt.Sprintf("eras[%d].", i)
		f.check(length(era.Name, 1, 80), p+"name", "O nome do período precisa ter de 1 a 80 caracteres.")
		f.check(!names[era.Name], p+"name", "Já existe um período com esse nome.")
		f.check(era.Start >= minYear && era.Start <= maxYear, p+"start", "Ano fora do intervalo.")
		f.check(era.End == nil || (*era.End >= era.Start && *era.End <= maxYear), p+"end", "O fim precisa ser depois do início.")
		f.check(era.Color == nil || era.Color.Valid(), p+"color", "Cor inválida.")
		names[era.Name] = true
	}
	if err := f.err(); err != nil {
		return err
	}
	// Guardados em ordem: a tela mostra as faixas e os grupos nessa ordem.
	slices.SortStableFunc(t.Eras, domain.CompareEras)
	return nil
}

// eraRemap diz o que fazer com o período dos eventos quando a lista muda: nome antigo
// para o novo (renomeado) ou nil (removido). Nomes que continuam iguais não entram.
// renamed liga nome novo a nome antigo; dá para trocar dois nomes de uma vez.
func eraRemap(before, after []domain.Era, renamed map[string]string) map[string]*string {
	kept := map[string]bool{}
	for _, e := range after {
		kept[e.Name] = true
	}
	renamedTo := map[string]string{}
	for to, from := range renamed {
		if kept[to] && to != from {
			renamedTo[from] = to
		}
	}
	out := map[string]*string{}
	for _, e := range before {
		if to, ok := renamedTo[e.Name]; ok {
			out[e.Name] = &to
		} else if _, other := renamed[e.Name]; !kept[e.Name] || other {
			// Sumiu, ou o nome agora é de outro período que foi renomeado para ele.
			out[e.Name] = nil
		}
	}
	return out
}

func normalizeTags(tags []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

type CreateTimelineInput struct {
	ActorID string
	TimelineFields
	Events []domain.EventData
}

type CreateTimeline struct{ d *Deps }

func NewCreateTimeline(d *Deps) *CreateTimeline { return &CreateTimeline{d} }

func (uc *CreateTimeline) Execute(ctx context.Context, in CreateTimelineInput) (TimelineDetailView, error) {
	if err := requireActor(in.ActorID); err != nil {
		return TimelineDetailView{}, err
	}
	now := uc.d.Clock.Now()
	t := domain.Timeline{
		ID: uc.d.IDs.NewID("tl"), OwnerID: in.ActorID, Color: domain.ColorA,
		Visibility: domain.VisibilityPublic, Contributions: domain.ContributionsSuggestions,
		Eras: []domain.Era{}, Tags: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	err := in.TimelineFields.apply(&t)
	f := fieldErrors{}
	if e, ok := err.(*domain.Error); ok {
		f = e.Fields
	} else if err != nil {
		return TimelineDetailView{}, err
	}
	f.check(len(in.Events) <= MaxImport, "events", "Crie a timeline com até 500 eventos e adicione o resto depois.")
	events := make([]domain.Event, len(in.Events))
	for i, data := range in.Events {
		data = normalizeEvent(data)
		validateEvent(f, fmt.Sprintf("events[%d].", i), data, t.Eras)
		events[i] = domain.Event{EventData: data, ID: uc.d.IDs.NewID("ev"), TimelineID: t.ID, AuthorID: in.ActorID, CreatedAt: now}
	}
	if err := f.err(); err != nil {
		return TimelineDetailView{}, err
	}
	err = uc.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		slug, err := uniqueSlug(ctx, uc.d, slugify(t.Title))
		if err != nil {
			return err
		}
		t.Slug = slug
		if err := uc.d.Timelines.Create(ctx, &t); err != nil {
			return err
		}
		return uc.d.Events.CreateMany(ctx, events)
	})
	if err != nil {
		return TimelineDetailView{}, err
	}
	return timelineDetail(ctx, uc.d, &t, in.ActorID)
}

func uniqueSlug(ctx context.Context, d *Deps, base string) (string, error) {
	slug := base
	for i := 2; ; i++ {
		taken, err := d.Timelines.SlugTaken(ctx, slug)
		if err != nil || !taken {
			return slug, err
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

type SlugInput struct {
	ActorID string
	Slug    string
}

type GetTimeline struct{ d *Deps }

func NewGetTimeline(d *Deps) *GetTimeline { return &GetTimeline{d} }

func (uc *GetTimeline) Execute(ctx context.Context, in SlugInput) (TimelineDetailView, error) {
	t, err := visibleTimeline(ctx, uc.d, in.Slug, in.ActorID)
	if err != nil {
		return TimelineDetailView{}, err
	}
	return timelineDetail(ctx, uc.d, t, in.ActorID)
}

type UpdateTimelineInput struct {
	ActorID string
	Slug    string
	TimelineFields
}

type UpdateTimeline struct{ d *Deps }

func NewUpdateTimeline(d *Deps) *UpdateTimeline { return &UpdateTimeline{d} }

func (uc *UpdateTimeline) Execute(ctx context.Context, in UpdateTimelineInput) (TimelineDetailView, error) {
	t, err := ownedTimeline(ctx, uc.d, in.Slug, in.ActorID)
	if err != nil {
		return TimelineDetailView{}, err
	}
	before := t.Eras
	if err := in.TimelineFields.apply(t); err != nil {
		return TimelineDetailView{}, err
	}
	t.UpdatedAt = uc.d.Clock.Now()
	remap := map[string]*string{}
	if in.Eras != nil {
		remap = eraRemap(before, t.Eras, in.RenamedEras)
	}
	err = uc.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.d.Timelines.Update(ctx, t); err != nil {
			return err
		}
		if len(remap) == 0 {
			return nil
		}
		// Eventos de um período removido ficam sem período; de um renomeado, seguem o nome novo.
		return uc.d.Events.RemapEras(ctx, t.ID, remap)
	})
	if err != nil {
		return TimelineDetailView{}, err
	}
	return timelineDetail(ctx, uc.d, t, in.ActorID)
}

type DeleteTimeline struct{ d *Deps }

func NewDeleteTimeline(d *Deps) *DeleteTimeline { return &DeleteTimeline{d} }

func (uc *DeleteTimeline) Execute(ctx context.Context, in SlugInput) (Empty, error) {
	t, err := ownedTimeline(ctx, uc.d, in.Slug, in.ActorID)
	if err != nil {
		return Empty{}, err
	}
	return Empty{}, uc.d.Timelines.Delete(ctx, t.ID)
}

// ListTags são as tags mais usadas, para os chips do explorar.
type ListTags struct{ d *Deps }

func NewListTags(d *Deps) *ListTags { return &ListTags{d} }

func (uc *ListTags) Execute(ctx context.Context, _ Empty) ([]domain.TagCount, error) {
	return uc.d.Timelines.Tags(ctx, 30)
}
