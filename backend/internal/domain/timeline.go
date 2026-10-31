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

// PeriodColor é a cor da faixa de um período. Sem cor, a tela escolhe pela posição.
type PeriodColor string

// PeriodColors na ordem em que a tela distribui as cores quando o período não tem uma.
var PeriodColors = []PeriodColor{"azul", "laranja", "verde", "roxo", "amarelo", "vermelho", "cinza"}

func (c PeriodColor) Valid() bool { return slices.Contains(PeriodColors, c) }

// Era é um período da timeline (ex.: República Velha, 1889–1930). Períodos podem se
// sobrepor: a Era Vargas (1930–1945) contém o Estado Novo (1937–1945). Sem End, o
// período continua até hoje.
type Era struct {
	Name  string
	Start int
	End   *int
	Color *PeriodColor
}

// CompareEras ordena os períodos pelo início e, no mesmo início, o mais longo primeiro
// (quem contém vem antes de quem está dentro). Sem fim conta como o mais longo.
func CompareEras(a, b Era) int {
	if a.Start != b.Start {
		return a.Start - b.Start
	}
	switch {
	case a.End == nil && b.End == nil:
		return 0
	case a.End == nil:
		return -1
	case b.End == nil:
		return 1
	}
	return *b.End - *a.End
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
