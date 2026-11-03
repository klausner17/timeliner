package usecase

import (
	"context"
	"log/slog"
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
	// Delete apaga uma sessão (sair). Apagar uma que não existe não é erro.
	Delete(ctx context.Context, tokenHash string) error
	// DeleteByUser apaga todas as sessões do usuário, menos a de exceptHash (pode ser "").
	DeleteByUser(ctx context.Context, userID, exceptHash string) error
}

type AuthTokenRepository interface {
	Create(ctx context.Context, t domain.AuthToken) error
	// Consume apaga o token e devolve o dono, se ele existir, for desse tipo e não
	// tiver vencido em now; senão devolve domain.ErrNotFound.
	Consume(ctx context.Context, tokenHash string, purpose domain.TokenPurpose, now time.Time) (string, error)
	// DeleteByUser apaga os tokens do usuário desse tipo (ex.: links antigos de senha).
	DeleteByUser(ctx context.Context, userID string, purpose domain.TokenPurpose) error
}

// Mailer manda e-mails.
type Mailer interface {
	Send(ctx context.Context, e domain.Email) error
}

// Paging é a página pedida a um repository.
type Paging struct {
	Limit  int
	Offset int
}

// SuggestionFilter escolhe as sugestões listadas. Os campos vazios não filtram.
type SuggestionFilter struct {
	TimelineID string
	AuthorID   string
	Status     domain.SuggestionStatus
	// ViewerID deixa só as sugestões em timelines que ele pode abrir.
	ViewerID string
	// OnlyVisible aplica o filtro de ViewerID mesmo sem login (ViewerID vazio):
	// assim quem não entrou só conta as sugestões em timelines públicas ou de link.
	OnlyVisible bool
	Paging
}

type TimelineSort string

const (
	SortPopular TimelineSort = "popular"
	SortRecent  TimelineSort = "recent"
)

// CalendarGregorian é o valor de TimelineFilter.Calendar para o calendário gregoriano.
const CalendarGregorian = "gregoriano"

// MaxListSlugs limita quantos slugs dá para pedir de uma vez na listagem.
const MaxListSlugs = 20

type TimelineFilter struct {
	Query    string
	Category string
	Tag      string
	OwnerID  string
	// OnlyOpen deixa só as que aceitam sugestões.
	OnlyOpen bool
	// IncludeUnlisted inclui rascunhos, privadas e "link" (só quando o dono está vendo as suas).
	IncludeUnlisted bool
	// ViewerID é quem está vendo. Com IncludeMine, as timelines dele entram junto com as
	// públicas; com Slugs, entram todas as que ele pode abrir (as de link também).
	ViewerID    string
	IncludeMine bool
	// Slugs restringe às timelines com esses slugs (ex.: as já escolhidas no comparar).
	Slugs []string
	// FollowedBy restringe às timelines que esse usuário segue, as seguidas mais
	// recentemente primeiro. Como Slugs, vale tudo que ViewerID pode abrir.
	FollowedBy string
	// FavoritedBy restringe às favoritas desse usuário, as favoritadas mais recentemente
	// primeiro. Vale tudo que ViewerID pode abrir, como FollowedBy.
	FavoritedBy string
	// Calendar restringe a um calendário: CalendarGregorian, "escala:<escala>" para as fixas
	// ou o ID de um calendário ficcional. Vazio não filtra.
	Calendar string
	// WithEvents deixa só as que têm pelo menos um evento.
	WithEvents bool
	Sort       TimelineSort
	Limit      int
	Offset     int
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
	// RecordView conta uma visualização de viewer no dia (AAAA-MM-DD). Repetir no mesmo
	// dia não conta de novo.
	RecordView(ctx context.Context, timelineID, viewer, day string) error
	// ByCalendar devolve todas as timelines que usam o calendário, de qualquer dono.
	ByCalendar(ctx context.Context, calendarID string) ([]domain.Timeline, error)
}

type CalendarRepository interface {
	Create(ctx context.Context, c *domain.Calendar) error
	Update(ctx context.Context, c *domain.Calendar) error
	Delete(ctx context.Context, id string) error
	ByID(ctx context.Context, id string) (*domain.Calendar, error)
	// ByIDs devolve os calendários encontrados, indexados por ID.
	ByIDs(ctx context.Context, ids []string) (map[string]domain.Calendar, error)
	// Usable devolve uma página dos calendários do usuário e dos compartilhados, por nome
	// (query filtra pelo nome), e o total.
	Usable(ctx context.Context, userID, query string, p Paging) ([]domain.Calendar, int, error)
	// InUse diz se alguma timeline usa o calendário.
	InUse(ctx context.Context, id string) (bool, error)
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
	// LatestYearIn devolve o maior ano de início ou de fim de evento das timelines que cai
	// em [from, to], ou nil se nenhum cai.
	LatestYearIn(ctx context.Context, timelineIDs []string, from, to int) (*int, error)
	// ShiftYears desloca o início e o fim dos eventos das timelines (domain.ShiftYear).
	ShiftYears(ctx context.Context, timelineIDs []string, shifts []domain.YearShift) error
}

type SuggestionRepository interface {
	Create(ctx context.Context, s *domain.Suggestion) error
	Update(ctx context.Context, s *domain.Suggestion) error
	ByID(ctx context.Context, id string) (*domain.Suggestion, error)
	// List devolve uma página, as mais recentes primeiro, e o total que bate com o filtro.
	List(ctx context.Context, f SuggestionFilter) ([]domain.Suggestion, int, error)
	// ShiftYears desloca os anos dos eventos propostos e anteriores (domain.ShiftYear).
	ShiftYears(ctx context.Context, timelineIDs []string, shifts []domain.YearShift) error
}

type FollowRepository interface {
	FollowTimeline(ctx context.Context, userID, timelineID string) error
	UnfollowTimeline(ctx context.Context, userID, timelineID string) error
	FollowUser(ctx context.Context, followerID, followedID string) error
	UnfollowUser(ctx context.Context, followerID, followedID string) error
	// FollowedUsers devolve uma página de IDs, os seguidos mais recentes primeiro, e o total.
	// As timelines seguidas vêm de TimelineRepository.List com FollowedBy.
	FollowedUsers(ctx context.Context, userID string, p Paging) ([]string, int, error)
	// FavoriteTimeline e UnfavoriteTimeline marcam e desmarcam a estrela. São idempotentes.
	// As favoritas vêm de TimelineRepository.List com FavoritedBy.
	FavoriteTimeline(ctx context.Context, userID, timelineID string) error
	UnfavoriteTimeline(ctx context.Context, userID, timelineID string) error
}

type ComparisonRepository interface {
	Create(ctx context.Context, c *domain.SavedComparison) error
	// ListByUser devolve uma página, as mais recentes primeiro, e o total. Com slugs, só a
	// comparação dessas timelines, nessa ordem.
	ListByUser(ctx context.Context, userID string, slugs []string, p Paging) ([]domain.SavedComparison, int, error)
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
// LinkPreviewer lê a prévia de uma página (título, resumo, miniatura). Falha quando a
// página não responde ou não é HTML; quem chama mostra só o endereço nesse caso.
type LinkPreviewer interface {
	Preview(ctx context.Context, url string) (domain.LinkPreview, error)
}

type Deps struct {
	Users       UserRepository
	Sessions    SessionRepository
	AuthTokens  AuthTokenRepository
	Timelines   TimelineRepository
	Calendars   CalendarRepository
	Events      EventRepository
	Suggestions SuggestionRepository
	Follows     FollowRepository
	Comparisons ComparisonRepository
	Tx          TxManager
	Hasher      PasswordHasher
	Clock       Clock
	IDs         IDGenerator
	// Previews monta a prévia dos links dos eventos; nil mostra só os endereços.
	Previews LinkPreviewer
	// SignupClosed fecha o POST /auth/register; contas novas so pelo comando create-user.
	SignupClosed bool
	// Mailer manda os e-mails de confirmação e de senha; nil não manda nada.
	Mailer Mailer
	// AppURL é o endereço do site (ex.: https://timeliner.com.br), base dos links nos e-mails.
	AppURL string
	// Log recebe erros que não devem chegar a quem pediu (ex.: e-mail que falhou); nil descarta.
	Log *slog.Logger
}
