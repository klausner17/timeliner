package usecase

import (
	"context"
	"time"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

// Os repositories devolvem domain.ErrNotFound quando não acham o registro
// e domain.ErrDuplicate quando uma chave única se repete.

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	Update(ctx context.Context, u *domain.User) error
	ByID(ctx context.Context, id string) (*domain.User, error)
	ByHandle(ctx context.Context, handle string) (*domain.User, error)
	ByEmail(ctx context.Context, email string) (*domain.User, error)
	// ByIDs devolve os usuários encontrados, indexados por ID.
	ByIDs(ctx context.Context, ids []string) (map[string]domain.User, error)
	Stats(ctx context.Context, userID, viewerID string) (domain.UserStats, error)
}

type SessionRepository interface {
	Create(ctx context.Context, s domain.Session) error
	// UserID devolve o dono de uma sessão que ainda não expirou em now.
	UserID(ctx context.Context, tokenHash string, now time.Time) (string, error)
}

type TimelineSort string

const (
	SortPopular TimelineSort = "popular"
	SortRecent  TimelineSort = "recent"
)

type TimelineFilter struct {
	Query    string
	Category string
	Tag      string
	OwnerID  string
	// OnlyOpen deixa só as que aceitam sugestões.
	OnlyOpen bool
	// IncludeUnlisted inclui rascunhos, privadas e "link" (só quando o dono está vendo as suas).
	IncludeUnlisted bool
	Sort            TimelineSort
	Limit           int
	Offset          int
}

type TimelineRepository interface {
	Create(ctx context.Context, t *domain.Timeline) error
	Update(ctx context.Context, t *domain.Timeline) error
	Delete(ctx context.Context, id string) error
	ByID(ctx context.Context, id string) (*domain.Timeline, error)
	BySlug(ctx context.Context, slug string) (*domain.Timeline, error)
	SlugTaken(ctx context.Context, slug string) (bool, error)
	// List devolve uma página e o total que bate com o filtro.
	List(ctx context.Context, f TimelineFilter) ([]domain.Timeline, int, error)
	// Stats calcula os números de várias timelines de uma vez.
	Stats(ctx context.Context, ids []string, viewerID string) (map[string]domain.TimelineStats, error)
	// Tags devolve as tags mais usadas nas timelines listadas.
	Tags(ctx context.Context, limit int) ([]domain.TagCount, error)
}

// EventQuery pede uma página de eventos de uma timeline, em ordem cronológica.
type EventQuery struct {
	TimelineID string
	// From e To filtram por eventos que tocam o intervalo (um período 1960–1970 entra em from=1965).
	From *int
	To   *int
	// After pula tudo até esse evento, inclusive (paginação por cursor).
	After *domain.EventCursor
	Limit int
}

// SummaryQuery divide [From, To] em Buckets faixas iguais de anos.
type SummaryQuery struct {
	TimelineID string
	From       int
	To         int
	Buckets    int
	// Sample é quantos eventos de exemplo vêm em cada faixa.
	Sample int
}

type EventRepository interface {
	Create(ctx context.Context, e *domain.Event) error
	// CreateMany grava vários eventos de uma vez (importação).
	CreateMany(ctx context.Context, es []domain.Event) error
	Update(ctx context.Context, e *domain.Event) error
	Delete(ctx context.Context, id string) error
	ByID(ctx context.Context, id string) (*domain.Event, error)
	// ListByTimeline devolve até q.Limit eventos em ordem cronológica.
	ListByTimeline(ctx context.Context, q EventQuery) ([]domain.Event, error)
	// Summary devolve só as faixas com eventos, em ordem, cada uma com a contagem e
	// os primeiros q.Sample eventos.
	Summary(ctx context.Context, q SummaryQuery) ([]domain.YearBucket, error)
	// RemapEras troca o período dos eventos da timeline: cada nome antigo vira o novo, ou
	// nenhum período quando o valor é nil. Tudo de uma vez, para trocas como A↔B.
	RemapEras(ctx context.Context, timelineID string, remap map[string]*string) error
	// Overlaps devolve até limit anos de [from, to] em que duas ou mais das timelines têm
	// evento começando, em ordem, e o total desses anos.
	Overlaps(ctx context.Context, timelineIDs []string, from, to, limit int) ([]domain.YearOverlap, int, error)
}

type SuggestionRepository interface {
	Create(ctx context.Context, s *domain.Suggestion) error
	Update(ctx context.Context, s *domain.Suggestion) error
	ByID(ctx context.Context, id string) (*domain.Suggestion, error)
	// ListByTimeline devolve as mais recentes primeiro; status vazio traz todas.
	ListByTimeline(ctx context.Context, timelineID string, status domain.SuggestionStatus) ([]domain.Suggestion, error)
	ListByAuthor(ctx context.Context, authorID string) ([]domain.Suggestion, error)
}

type FollowRepository interface {
	FollowTimeline(ctx context.Context, userID, timelineID string) error
	UnfollowTimeline(ctx context.Context, userID, timelineID string) error
	FollowUser(ctx context.Context, followerID, followedID string) error
	UnfollowUser(ctx context.Context, followerID, followedID string) error
	// FollowedTimelines e FollowedUsers devolvem IDs, os mais recentes primeiro.
	FollowedTimelines(ctx context.Context, userID string) ([]string, error)
	FollowedUsers(ctx context.Context, userID string) ([]string, error)
}

type ComparisonRepository interface {
	Create(ctx context.Context, c *domain.SavedComparison) error
	// ListByUser devolve as mais recentes primeiro.
	ListByUser(ctx context.Context, userID string) ([]domain.SavedComparison, error)
	// Delete só apaga se a comparação for do usuário; senão devolve ErrNotFound.
	Delete(ctx context.Context, id, userID string) error
}

// TxManager roda fn numa transação: tudo que os repositories fazem com o ctx
// recebido é confirmado junto ou desfeito junto.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// PasswordHasher esconde o algoritmo de senha (bcrypt em produção).
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

// Clock e IDGenerator existem para os testes controlarem tempo e IDs.
type Clock interface{ Now() time.Time }

type IDGenerator interface {
	// NewID devolve um ID novo com o prefixo dado (ex.: "tl").
	NewID(prefix string) string
	// NewToken devolve um token aleatório para sessões.
	NewToken() string
}

// Deps junta tudo que os casos de uso podem precisar. Cada caso de uso usa só uma parte.
type Deps struct {
	Users       UserRepository
	Sessions    SessionRepository
	Timelines   TimelineRepository
	Events      EventRepository
	Suggestions SuggestionRepository
	Follows     FollowRepository
	Comparisons ComparisonRepository
	Tx          TxManager
	Hasher      PasswordHasher
	Clock       Clock
	IDs         IDGenerator
	// SignupClosed fecha o POST /auth/register; contas novas so pelo comando create-user.
	SignupClosed bool
}
