package domain

import "time"

type SuggestionKind string

const (
	SuggestionNewEvent       SuggestionKind = "new_event"
	SuggestionDescriptionFix SuggestionKind = "description_fix"
	SuggestionDateFix        SuggestionKind = "date_fix"
)

func (k SuggestionKind) Valid() bool {
	return k == SuggestionNewEvent || k == SuggestionDescriptionFix || k == SuggestionDateFix
}

// IsFix diz se a sugestão corrige um evento que já existe.
func (k SuggestionKind) IsFix() bool { return k == SuggestionDescriptionFix || k == SuggestionDateFix }

type SuggestionStatus string

const (
	StatusPending          SuggestionStatus = "pendente"
	StatusApproved         SuggestionStatus = "aprovada"
	StatusRejected         SuggestionStatus = "recusada"
	StatusChangesRequested SuggestionStatus = "ajuste"
)

func (s SuggestionStatus) Valid() bool {
	switch s {
	case StatusPending, StatusApproved, StatusRejected, StatusChangesRequested:
		return true
	}
	return false
}

type Suggestion struct {
	ID         string
	TimelineID string
	Kind       SuggestionKind
	Status     SuggestionStatus
	AuthorID   string
	// EventID é o evento corrigido, nas correções.
	EventID  *string
	Proposed EventData
	Note     *string
	Feedback *string
	// AppliedEventID é o evento criado ou corrigido depois da aprovação.
	AppliedEventID *string
	// Previous guarda o evento como era antes de uma correção aprovada, para desfazer.
	Previous   *Event
	CreatedAt  time.Time
	ReviewedAt *time.Time
}
