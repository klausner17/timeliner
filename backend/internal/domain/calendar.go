package domain

import "time"

// Calendar é a contagem de anos de um mundo ficcional (ex.: Terra-média), usada por uma
// ou mais timelines. É uma lista de eras em sequência; em cada era os anos começam do 1.
//
// Por baixo, os eventos continuam com um ano contínuo (Event.Year): o ano 1 da primeira
// era é o ano 1, e cada era começa logo depois do fim da anterior. A tela converte esse
// ano em "T.A. 3019" com as eras do calendário.
type Calendar struct {
	ID          string
	Name        string
	Description string
	OwnerID     string
	// Shared deixa qualquer pessoa usar o calendário nas próprias timelines. Só o dono edita.
	Shared    bool
	Eras      []CalendarEra
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CalendarEra é uma era de contagem: Terceira Era (T.A.), 3021 anos. Só a última pode
// ficar sem duração (continua indefinidamente).
type CalendarEra struct {
	ID    string
	Name  string
	Abbr  string
	Years *int
}

// OpenEraYears é até onde vai, no eixo, uma era sem duração (a última do calendário).
const OpenEraYears = 100000

// EraSpan é o trecho do ano contínuo que uma era ocupa: de Start a End, inclusive.
type EraSpan struct {
	ID    string
	Start int
	End   int
}

// Spans devolve o trecho de cada era, na ordem do calendário.
func (c *Calendar) Spans() []EraSpan {
	out := make([]EraSpan, len(c.Eras))
	start := 1
	for i, e := range c.Eras {
		n := OpenEraYears
		if e.Years != nil {
			n = *e.Years
		}
		out[i] = EraSpan{ID: e.ID, Start: start, End: start + n - 1}
		start += n
	}
	return out
}

// Bounds devolve o primeiro e o último ano contínuo do calendário.
func (c *Calendar) Bounds() (int, int) {
	s := c.Spans()
	if len(s) == 0 {
		return 1, 0
	}
	return s[0].Start, s[len(s)-1].End
}

// YearShift diz que os anos de From a To (inclusive) andam Delta anos.
type YearShift struct {
	From  int
	To    int
	Delta int
}

// ShiftYear devolve o ano depois dos deslocamentos (o primeiro que contém o ano vale).
func ShiftYear(year int, shifts []YearShift) int {
	for _, s := range shifts {
		if year >= s.From && year <= s.To {
			return year + s.Delta
		}
	}
	return year
}
