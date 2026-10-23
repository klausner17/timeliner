package domain

import "time"

// EventData são os campos que o usuário escreve num evento.
type EventData struct {
	Year        int
	EndYear     *int
	Date        *string
	Title       string
	Description string
	SourceURL   *string
	Era         *string
}

type Event struct {
	EventData
	ID         string
	TimelineID string
	AuthorID   string
	CreatedAt  time.Time
}

// EventCursor é a posição de um evento na ordem cronológica (ano, fim, criação, ID).
// Serve para pedir a página seguinte a partir dele.
type EventCursor struct {
	Year      int
	EndYear   int
	CreatedAt time.Time
	ID        string
}

// CursorOf devolve a posição do evento na ordem cronológica.
func CursorOf(e Event) EventCursor {
	end := e.Year
	if e.EndYear != nil {
		end = *e.EndYear
	}
	return EventCursor{Year: e.Year, EndYear: end, CreatedAt: e.CreatedAt, ID: e.ID}
}

// YearBucket é uma faixa de anos com quantos eventos começam nela e os primeiros deles.
type YearBucket struct {
	From   int
	To     int
	Count  int
	Sample []Event
}

// BucketOf diz em qual das n faixas iguais de [from, from+span) cai o ano. Eventos que
// começam antes de from (mas tocam o intervalo) entram na primeira faixa.
func BucketOf(year, from, span, n int) int {
	return int(int64(max(year, from)-from) * int64(n) / int64(span))
}

// BucketRange devolve o primeiro e o último ano da faixa i, na mesma conta de BucketOf.
func BucketRange(i, from, span, n int) (int, int) {
	lo := (int64(i)*int64(span) + int64(n) - 1) / int64(n)
	hi := (int64(i+1)*int64(span)+int64(n)-1)/int64(n) - 1
	return from + int(lo), from + int(hi)
}

// YearOverlap é um ano em que mais de uma timeline tem evento.
type YearOverlap struct {
	Year        int
	TimelineIDs []string
}
