package domain

import "time"

// SavedComparison é uma comparação que o usuário guardou para abrir depois.
type SavedComparison struct {
	ID        string
	UserID    string
	Slugs     []string
	Start     int
	End       int
	CreatedAt time.Time
}

// TagCount é uma tag e em quantas timelines públicas ela aparece.
type TagCount struct {
	Tag   string
	Count int
}
