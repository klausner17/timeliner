package usecase

import (
	"context"
	"strings"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

type ListSuggestionsInput struct {
	ActorID string
	Slug    string
	Status  domain.SuggestionStatus
}

type ListSuggestions struct{ d *Deps }

func NewListSuggestions(d *Deps) *ListSuggestions { return &ListSuggestions{d} }

func (uc *ListSuggestions) Execute(ctx context.Context, in ListSuggestionsInput) ([]SuggestionView, error) {
	if in.Status != "" && !in.Status.Valid() {
		return nil, domain.Invalid(map[string]string{"status": "Status inválido."})
	}
	t, err := visibleTimeline(ctx, uc.d, in.Slug, in.ActorID)
	if err != nil {
		return nil, err
	}
	ss, err := uc.d.Suggestions.ListByTimeline(ctx, t.ID, in.Status)
	if err != nil {
		return nil, err
	}
	return suggestionViews(ctx, uc.d, ss, in.ActorID)
}

type CreateSuggestionInput struct {
	ActorID string
	Slug    string
	Kind    domain.SuggestionKind
	EventID *string
	Event   domain.EventData
	Note    *string
}

// CreateSuggestion é a tela contribuir: propõe um evento novo ou uma correção.
type CreateSuggestion struct{ d *Deps }

func NewCreateSuggestion(d *Deps) *CreateSuggestion { return &CreateSuggestion{d} }

func (uc *CreateSuggestion) Execute(ctx context.Context, in CreateSuggestionInput) (SuggestionView, error) {
	if err := requireActor(in.ActorID); err != nil {
		return SuggestionView{}, err
	}
	t, err := visibleTimeline(ctx, uc.d, in.Slug, in.ActorID)
	if err != nil {
		return SuggestionView{}, err
	}
	if t.Contributions != domain.ContributionsSuggestions {
		return SuggestionView{}, domain.Forbidden("Esta timeline não está aceitando sugestões.")
	}
	f := fieldErrors{}
	f.check(in.Kind.Valid(), "kind", "Tipo de sugestão inválido.")
	data := normalizeEvent(in.Event)
	validateEvent(f, "event.", data, t.Eras)
	note := emptyToNil(in.Note)
	if note != nil {
		f.check(length(*note, 0, 1000), "note", "O recado pode ter até 1000 caracteres.")
	}
	var eventID *string
	if in.Kind.IsFix() {
		if in.EventID == nil || *in.EventID == "" {
			f.add("eventId", "Escolha o evento que você quer corrigir.")
		} else if _, err := eventOf(ctx, uc.d, t, *in.EventID); err != nil {
			if domain.KindOf(err) != domain.KindNotFound {
				return SuggestionView{}, err
			}
			f.add("eventId", "Esse evento não existe nesta timeline.")
		} else {
			eventID = in.EventID
		}
	}
	if err := f.err(); err != nil {
		return SuggestionView{}, err
	}
	s := domain.Suggestion{
		ID: uc.d.IDs.NewID("sg"), TimelineID: t.ID, Kind: in.Kind, Status: domain.StatusPending,
		AuthorID: in.ActorID, EventID: eventID, Proposed: data, Note: note, CreatedAt: uc.d.Clock.Now(),
	}
	if err := uc.d.Suggestions.Create(ctx, &s); err != nil {
		return SuggestionView{}, err
	}
	return suggestionView(ctx, uc.d, &s, in.ActorID)
}

type ResubmitSuggestionInput struct {
	ActorID      string
	SuggestionID string
	Event        *domain.EventData
	Note         *string
}

// ResubmitSuggestion é quem sugeriu mandando de novo depois de um pedido de ajuste.
type ResubmitSuggestion struct{ d *Deps }

func NewResubmitSuggestion(d *Deps) *ResubmitSuggestion { return &ResubmitSuggestion{d} }

func (uc *ResubmitSuggestion) Execute(ctx context.Context, in ResubmitSuggestionInput) (SuggestionView, error) {
	if err := requireActor(in.ActorID); err != nil {
		return SuggestionView{}, err
	}
	s, t, err := suggestionAndTimeline(ctx, uc.d, in.SuggestionID, in.ActorID)
	if err != nil {
		return SuggestionView{}, err
	}
	if s.AuthorID != in.ActorID {
		return SuggestionView{}, domain.Forbidden("Só quem enviou a sugestão pode editá-la.")
	}
	if s.Status != domain.StatusPending && s.Status != domain.StatusChangesRequested {
		return SuggestionView{}, domain.Conflict("Esta sugestão já foi revisada e não pode mais ser editada.")
	}
	f := fieldErrors{}
	if in.Event != nil {
		s.Proposed = normalizeEvent(*in.Event)
		validateEvent(f, "event.", s.Proposed, t.Eras)
	}
	if in.Note != nil {
		s.Note = emptyToNil(in.Note)
		if s.Note != nil {
			f.check(length(*s.Note, 0, 1000), "note", "O recado pode ter até 1000 caracteres.")
		}
	}
	if err := f.err(); err != nil {
		return SuggestionView{}, err
	}
	s.Status = domain.StatusPending
	if err := uc.d.Suggestions.Update(ctx, s); err != nil {
		return SuggestionView{}, err
	}
	return suggestionView(ctx, uc.d, s, in.ActorID)
}

type ReviewDecision string

const (
	DecisionApprove        ReviewDecision = "approve"
	DecisionReject         ReviewDecision = "reject"
	DecisionRequestChanges ReviewDecision = "request_changes"
)

type ReviewSuggestionInput struct {
	ActorID      string
	SuggestionID string
	Decision     ReviewDecision
	Feedback     *string
}

// ReviewSuggestion aprova, recusa ou pede ajuste. Aprovar aplica a mudança na timeline
// na mesma transação, guardando o evento anterior nas correções.
type ReviewSuggestion struct{ d *Deps }

func NewReviewSuggestion(d *Deps) *ReviewSuggestion { return &ReviewSuggestion{d} }

func (uc *ReviewSuggestion) Execute(ctx context.Context, in ReviewSuggestionInput) (SuggestionView, error) {
	s, t, err := reviewable(ctx, uc.d, in.SuggestionID, in.ActorID)
	if err != nil {
		return SuggestionView{}, err
	}
	feedback := emptyToNil(in.Feedback)
	f := fieldErrors{}
	switch in.Decision {
	case DecisionApprove, DecisionReject:
	case DecisionRequestChanges:
		f.check(feedback != nil, "feedback", "Diga o que precisa mudar.")
	default:
		f.add("decision", "Decisão inválida.")
	}
	if feedback != nil {
		f.check(length(*feedback, 0, 1000), "feedback", "O retorno pode ter até 1000 caracteres.")
	}
	if err := f.err(); err != nil {
		return SuggestionView{}, err
	}
	if s.Status != domain.StatusPending {
		return SuggestionView{}, domain.Conflict("Esta sugestão não está mais pendente.")
	}
	now := uc.d.Clock.Now()
	s.Feedback, s.ReviewedAt = feedback, &now
	err = uc.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		switch in.Decision {
		case DecisionReject:
			s.Status = domain.StatusRejected
		case DecisionRequestChanges:
			s.Status = domain.StatusChangesRequested
		case DecisionApprove:
			s.Status = domain.StatusApproved
			if err := apply(ctx, uc.d, s, t); err != nil {
				return err
			}
		}
		return uc.d.Suggestions.Update(ctx, s)
	})
	if err != nil {
		return SuggestionView{}, err
	}
	return suggestionView(ctx, uc.d, s, in.ActorID)
}

// apply leva a sugestão aprovada para a timeline.
func apply(ctx context.Context, d *Deps, s *domain.Suggestion, t *domain.Timeline) error {
	f := fieldErrors{}
	validateEvent(f, "event.", s.Proposed, t.Eras)
	if len(f) > 0 {
		// As eras da timeline podem ter mudado desde que a sugestão foi enviada.
		return domain.Conflict("A sugestão não combina mais com a timeline (o período pode ter sido removido). Peça um ajuste.")
	}
	if s.Kind == domain.SuggestionNewEvent {
		e := domain.Event{EventData: s.Proposed, ID: d.IDs.NewID("ev"), TimelineID: t.ID, AuthorID: s.AuthorID, CreatedAt: d.Clock.Now()}
		if err := d.Events.Create(ctx, &e); err != nil {
			return err
		}
		s.AppliedEventID = &e.ID
		return touch(ctx, d, t)
	}
	e, err := eventOf(ctx, d, t, deref(s.EventID))
	if err != nil {
		if domain.KindOf(err) == domain.KindNotFound {
			return domain.Conflict("O evento desta correção foi apagado.")
		}
		return err
	}
	previous := *e
	s.Previous = &previous
	e.EventData = s.Proposed
	if err := d.Events.Update(ctx, e); err != nil {
		return err
	}
	s.AppliedEventID = &e.ID
	return touch(ctx, d, t)
}

type SuggestionRefInput struct {
	ActorID      string
	SuggestionID string
}

// ReopenSuggestion desfaz a revisão e volta a sugestão para pendente. Se estava
// aprovada, remove o evento criado ou devolve o evento ao estado anterior.
type ReopenSuggestion struct{ d *Deps }

func NewReopenSuggestion(d *Deps) *ReopenSuggestion { return &ReopenSuggestion{d} }

func (uc *ReopenSuggestion) Execute(ctx context.Context, in SuggestionRefInput) (SuggestionView, error) {
	s, t, err := reviewable(ctx, uc.d, in.SuggestionID, in.ActorID)
	if err != nil {
		return SuggestionView{}, err
	}
	if s.Status == domain.StatusPending {
		return SuggestionView{}, domain.Conflict("Esta sugestão já está pendente.")
	}
	err = uc.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if s.Status == domain.StatusApproved {
			if err := undo(ctx, uc.d, s, t); err != nil {
				return err
			}
		}
		s.Status, s.Feedback, s.ReviewedAt, s.AppliedEventID, s.Previous = domain.StatusPending, nil, nil, nil, nil
		return uc.d.Suggestions.Update(ctx, s)
	})
	if err != nil {
		return SuggestionView{}, err
	}
	return suggestionView(ctx, uc.d, s, in.ActorID)
}

func undo(ctx context.Context, d *Deps, s *domain.Suggestion, t *domain.Timeline) error {
	if s.AppliedEventID == nil {
		return nil
	}
	e, err := eventOf(ctx, d, t, *s.AppliedEventID)
	if domain.KindOf(err) == domain.KindNotFound {
		// O dono já apagou o evento: não há o que desfazer.
		return nil
	}
	if err != nil {
		return err
	}
	if s.Kind == domain.SuggestionNewEvent {
		err = d.Events.Delete(ctx, e.ID)
	} else if s.Previous != nil {
		e.EventData = s.Previous.EventData
		err = d.Events.Update(ctx, e)
	}
	if err != nil {
		return err
	}
	return touch(ctx, d, t)
}

func suggestionAndTimeline(ctx context.Context, d *Deps, id, actorID string) (*domain.Suggestion, *domain.Timeline, error) {
	s, err := d.Suggestions.ByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, nil, notFound(err, "Sugestão não encontrada.")
	}
	t, err := d.Timelines.ByID(ctx, s.TimelineID)
	if err == nil && !t.VisibleTo(actorID) {
		err = domain.ErrNotFound
	}
	if err != nil {
		return nil, nil, notFound(err, "Sugestão não encontrada.")
	}
	return s, t, nil
}

// reviewable carrega a sugestão e exige que o actor seja o dono da timeline.
func reviewable(ctx context.Context, d *Deps, id, actorID string) (*domain.Suggestion, *domain.Timeline, error) {
	if err := requireActor(actorID); err != nil {
		return nil, nil, err
	}
	s, t, err := suggestionAndTimeline(ctx, d, id, actorID)
	if err != nil {
		return nil, nil, err
	}
	if t.OwnerID != actorID {
		return nil, nil, domain.Forbidden("Só o dono da timeline pode revisar sugestões.")
	}
	return s, t, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
