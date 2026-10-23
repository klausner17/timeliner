package usecase

import (
	"context"
	"slices"
	"strings"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

type CompareInput struct {
	ActorID string
	Slugs   []string
	Start   *int
	End     *int
}

type CompareLane struct {
	Color    domain.TrackColor
	Timeline TimelineView
	Events   []EventView
}

type Overlap struct {
	Year int
	// Slugs das trilhas que têm evento nesse ano.
	Slugs []string
}

type ComparisonView struct {
	Start    int
	End      int
	Lanes    []CompareLane
	Overlaps []Overlap
}

// Compare põe de 2 a 4 timelines no mesmo eixo. Sem start/end, o eixo vai do primeiro
// ao último evento de todas elas.
type Compare struct{ d *Deps }

func NewCompare(d *Deps) *Compare { return &Compare{d} }

func (uc *Compare) Execute(ctx context.Context, in CompareInput) (ComparisonView, error) {
	slugs := []string{}
	for _, s := range in.Slugs {
		if s = strings.TrimSpace(s); s != "" && !slices.Contains(slugs, s) {
			slugs = append(slugs, s)
		}
	}
	f := fieldErrors{}
	f.check(len(slugs) >= 2 && len(slugs) <= 4, "slugs", "Escolha de 2 a 4 timelines diferentes.")
	f.check(in.Start == nil || in.End == nil || *in.Start <= *in.End, "end", "O ano final precisa ser depois do inicial.")
	if err := f.err(); err != nil {
		return ComparisonView{}, err
	}
	ts := make([]domain.Timeline, len(slugs))
	for i, slug := range slugs {
		t, err := visibleTimeline(ctx, uc.d, slug, in.ActorID)
		if err != nil {
			return ComparisonView{}, err
		}
		ts[i] = *t
	}
	views, err := timelineViews(ctx, uc.d, ts, in.ActorID)
	if err != nil {
		return ComparisonView{}, err
	}
	start, end := 0, 0
	first := true
	for _, v := range views {
		if v.Stats.StartYear == nil {
			continue
		}
		if first || *v.Stats.StartYear < start {
			start = *v.Stats.StartYear
		}
		if first || *v.Stats.EndYear > end {
			end = *v.Stats.EndYear
		}
		first = false
	}
	if in.Start != nil {
		start = *in.Start
	}
	if in.End != nil {
		end = *in.End
	}
	if end < start {
		end = start
	}
	out := ComparisonView{Start: start, End: end, Lanes: make([]CompareLane, len(views)), Overlaps: []Overlap{}}
	byYear := map[int][]string{}
	for i, v := range views {
		events, err := uc.d.Events.ListByTimeline(ctx, v.Timeline.ID, &start, &end)
		if err != nil {
			return ComparisonView{}, err
		}
		evs, err := eventViews(ctx, uc.d, events)
		if err != nil {
			return ComparisonView{}, err
		}
		out.Lanes[i] = CompareLane{Color: domain.TrackColors[i], Timeline: v, Events: evs}
		for _, e := range events {
			if !slices.Contains(byYear[e.Year], v.Timeline.Slug) {
				byYear[e.Year] = append(byYear[e.Year], v.Timeline.Slug)
			}
		}
	}
	for year, s := range byYear {
		if len(s) >= 2 {
			out.Overlaps = append(out.Overlaps, Overlap{Year: year, Slugs: s})
		}
	}
	slices.SortFunc(out.Overlaps, func(a, b Overlap) int { return a.Year - b.Year })
	return out, nil
}

type SaveComparisonInput struct {
	ActorID string
	Slugs   []string
	Start   int
	End     int
}

type SaveComparison struct{ d *Deps }

func NewSaveComparison(d *Deps) *SaveComparison { return &SaveComparison{d} }

func (uc *SaveComparison) Execute(ctx context.Context, in SaveComparisonInput) (domain.SavedComparison, error) {
	if err := requireActor(in.ActorID); err != nil {
		return domain.SavedComparison{}, err
	}
	f := fieldErrors{}
	f.check(len(in.Slugs) >= 2 && len(in.Slugs) <= 4, "timelines", "Escolha de 2 a 4 timelines.")
	f.check(in.Start <= in.End, "end", "O ano final precisa ser depois do inicial.")
	seen := map[string]bool{}
	for _, slug := range in.Slugs {
		f.check(!seen[slug], "timelines", "Não repita a mesma timeline.")
		seen[slug] = true
		if _, err := visibleTimeline(ctx, uc.d, slug, in.ActorID); err != nil {
			if domain.KindOf(err) != domain.KindNotFound {
				return domain.SavedComparison{}, err
			}
			f.add("timelines", "Uma das timelines não existe mais.")
		}
	}
	if err := f.err(); err != nil {
		return domain.SavedComparison{}, err
	}
	c := domain.SavedComparison{
		ID: uc.d.IDs.NewID("cmp"), UserID: in.ActorID, Slugs: in.Slugs, Start: in.Start, End: in.End, CreatedAt: uc.d.Clock.Now(),
	}
	if err := uc.d.Comparisons.Create(ctx, &c); err != nil {
		return domain.SavedComparison{}, err
	}
	return c, nil
}

type ListSavedComparisons struct{ d *Deps }

func NewListSavedComparisons(d *Deps) *ListSavedComparisons { return &ListSavedComparisons{d} }

func (uc *ListSavedComparisons) Execute(ctx context.Context, in ActorInput) ([]domain.SavedComparison, error) {
	if err := requireActor(in.ActorID); err != nil {
		return nil, err
	}
	return uc.d.Comparisons.ListByUser(ctx, in.ActorID)
}

type DeleteSavedComparisonInput struct {
	ActorID      string
	ComparisonID string
}

type DeleteSavedComparison struct{ d *Deps }

func NewDeleteSavedComparison(d *Deps) *DeleteSavedComparison { return &DeleteSavedComparison{d} }

func (uc *DeleteSavedComparison) Execute(ctx context.Context, in DeleteSavedComparisonInput) (Empty, error) {
	if err := requireActor(in.ActorID); err != nil {
		return Empty{}, err
	}
	return Empty{}, notFound(uc.d.Comparisons.Delete(ctx, in.ComparisonID, in.ActorID), "Comparação não encontrada.")
}
