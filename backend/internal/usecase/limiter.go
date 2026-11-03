package usecase

import (
	"sync"
	"time"
)

// limiter conta tentativas por chave (e-mail, IP) numa janela fixa. Fica na
// memória do processo: reiniciar o servidor zera as contagens, o que basta
// contra tentativa e erro e cadastro em massa num servidor só.
type limiter struct {
	max    int
	window time.Duration

	mu   sync.Mutex
	hits map[string]limiterEntry
}

type limiterEntry struct {
	count int
	since time.Time
}

func newLimiter(max int, window time.Duration) *limiter {
	return &limiter{max: max, window: window, hits: map[string]limiterEntry{}}
}

// Blocked diz se key já esgotou as tentativas da janela atual. Chave vazia
// (ex.: IP desconhecido) nunca bloqueia.
func (l *limiter) Blocked(key string, now time.Time) bool {
	if key == "" {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.hits[key]
	if ok && now.Sub(e.since) >= l.window {
		delete(l.hits, key)
		return false
	}
	return e.count >= l.max
}

// Hit conta uma tentativa de key.
func (l *limiter) Hit(key string, now time.Time) {
	if key == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	// Chaves inventadas também entram: de tempos em tempos, tira as vencidas.
	if len(l.hits) >= 10_000 {
		for k, e := range l.hits {
			if now.Sub(e.since) >= l.window {
				delete(l.hits, k)
			}
		}
	}
	e := l.hits[key]
	if e.count == 0 || now.Sub(e.since) >= l.window {
		e = limiterEntry{since: now}
	}
	e.count++
	l.hits[key] = e
}

// Reset esquece as tentativas de key.
func (l *limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}
