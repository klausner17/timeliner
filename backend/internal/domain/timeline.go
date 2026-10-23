package domain

import (
	"slices"
	"time"
)

type TrackColor string

const (
	ColorA TrackColor = "a"
	ColorB TrackColor = "b"
	ColorC TrackColor = "c"
	ColorD TrackColor = "d"
)

// TrackColors na ordem em que as trilhas de uma comparação recebem as cores.
var TrackColors = []TrackColor{ColorA, ColorB, ColorC, ColorD}

func (c TrackColor) Valid() bool { return slices.Contains(TrackColors, c) }

type Visibility string

const (
	VisibilityPublic  Visibility = "publica"
	VisibilityLink    Visibility = "link"
	VisibilityPrivate Visibility = "privada"
)

func (v Visibility) Valid() bool {
	return v == VisibilityPublic || v == VisibilityLink || v == VisibilityPrivate
}

type ContributionMode string

const (
	ContributionsSuggestions ContributionMode = "sugestoes"
	ContributionsInvited     ContributionMode = "convidados"
	ContributionsClosed      ContributionMode = "fechada"
)

func (m ContributionMode) Valid() bool {
	return m == ContributionsSuggestions || m == ContributionsInvited || m == ContributionsClosed
}

var Categories = []string{"História", "Política", "Esporte", "Ciência e tecnologia", "Cultura", "Pessoal"}

func ValidCategory(c string) bool { return slices.Contains(Categories, c) }

type Era struct {
	Name  string
	Start int
	End   *int
}

type Timeline struct {
	ID            string
	Slug          string
	Title         string
	Description   string
	Category      string
	Color         TrackColor
	OwnerID       string
	Visibility    Visibility
	Contributions ContributionMode
	Draft         bool
	Eras          []Era
	Tags          []string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// VisibleTo diz se o usuário pode abrir a timeline. Rascunhos e privadas só o dono vê;
// "link" qualquer um que tenha o slug.
func (t *Timeline) VisibleTo(userID string) bool {
	if t.OwnerID == userID {
		return true
	}
	return !t.Draft && t.Visibility != VisibilityPrivate
}

// Listed diz se a timeline aparece no explorar para quem não é o dono.
func (t *Timeline) Listed() bool { return !t.Draft && t.Visibility == VisibilityPublic }

// TimelineStats são os números calculados a partir de eventos, sugestões e seguidores.
type TimelineStats struct {
	EventCount         int
	ContributorCount   int
	PendingSuggestions int
	Followers          int
	Following          bool
	// Positions são as posições distintas dos anos de início no eixo StartYear–EndYear,
	// em milésimos (0 a 1000) e em ordem. Ficam no máximo 1001 valores, por maior que
	// seja a timeline.
	Positions []int
	StartYear *int
	EndYear   *int
}
