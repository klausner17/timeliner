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
