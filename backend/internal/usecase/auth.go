package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

const sessionTTL = 30 * 24 * time.Hour

type AuthOutput struct {
	Token string
	User  domain.User
}

// HashToken é como o token fica guardado: o banco nunca vê o token em si.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newSession(ctx context.Context, d *Deps, userID string) (string, error) {
	token := d.IDs.NewToken()
	now := d.Clock.Now()
	err := d.Sessions.Create(ctx, domain.Session{
		TokenHash: HashToken(token), UserID: userID, CreatedAt: now, ExpiresAt: now.Add(sessionTTL),
	})
	return token, err
}

type RegisterInput struct {
	Handle   string
	Name     string
	Email    string
	Password string
}

// Register cria a conta e já devolve uma sessão.
type Register struct {
	d *Deps
	// ignoreClosed deixa o comando create-user criar contas com o cadastro fechado.
	ignoreClosed bool
}

func NewRegister(d *Deps) *Register { return &Register{d: d} }

// NewAdminRegister cria contas mesmo com o cadastro fechado (comando create-user).
func NewAdminRegister(d *Deps) *Register { return &Register{d: d, ignoreClosed: true} }

func (uc *Register) Execute(ctx context.Context, in RegisterInput) (AuthOutput, error) {
	if uc.d.SignupClosed && !uc.ignoreClosed {
		return AuthOutput{}, domain.Forbidden("O cadastro está fechado no momento.")
	}
	in.Handle = strings.TrimSpace(in.Handle)
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	f := fieldErrors{}
	f.check(handleRe.MatchString(in.Handle), "handle", "Use de 3 a 30 letras minúsculas, números ou _.")
	f.check(in.Handle != "me", "handle", "Nome reservado.")
	f.check(length(in.Name, 1, 80), "name", "O nome precisa ter de 1 a 80 caracteres.")
	f.check(validEmail(in.Email), "email", "E-mail inválido.")
	f.check(len(in.Password) >= 8, "password", "A senha precisa ter pelo menos 8 caracteres.")
	if err := f.err(); err != nil {
		return AuthOutput{}, err
	}
	hash, err := uc.d.Hasher.Hash(in.Password)
	if err != nil {
		return AuthOutput{}, err
	}
	u := domain.User{
		ID: uc.d.IDs.NewID("usr"), Handle: in.Handle, Name: in.Name, Email: in.Email,
		PasswordHash: hash, Tint: domain.TintNeutral, CreatedAt: uc.d.Clock.Now(),
	}
	var token string
	err = uc.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := uc.d.Users.ByHandle(ctx, u.Handle); err == nil {
			return &domain.Error{Kind: domain.KindConflict, Message: "Esse nome de usuário já está em uso.", Fields: map[string]string{"handle": "já existe alguém com esse nome"}}
		}
		if _, err := uc.d.Users.ByEmail(ctx, u.Email); err == nil {
			return &domain.Error{Kind: domain.KindConflict, Message: "Já existe uma conta com esse e-mail.", Fields: map[string]string{"email": "já existe uma conta com esse e-mail"}}
		}
		if err := uc.d.Users.Create(ctx, &u); err != nil {
			if errors.Is(err, domain.ErrDuplicate) {
				return domain.Conflict("Nome de usuário ou e-mail já em uso.")
			}
			return err
		}
		token, err = newSession(ctx, uc.d, u.ID)
		return err
	})
	if err != nil {
		return AuthOutput{}, err
	}
	return AuthOutput{Token: token, User: u}, nil
}

type LoginInput struct {
	Email    string
	Password string
}

// Limite de senhas erradas por e-mail, contra tentativa e erro numa conta. Por
// e-mail e não por IP: atrás do proxy o IP do cliente não é confiável.
const (
	loginMaxFailures = 10
	loginWindow      = 15 * time.Minute
)

type loginFailures struct {
	count int
	since time.Time
}

type Login struct {
	d        *Deps
	mu       sync.Mutex
	failures map[string]loginFailures
}

func NewLogin(d *Deps) *Login { return &Login{d: d, failures: map[string]loginFailures{}} }

func (uc *Login) Execute(ctx context.Context, in LoginInput) (AuthOutput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	now := uc.d.Clock.Now()
	if uc.blocked(email, now) {
		return AuthOutput{}, domain.Unauthorized("Muitas tentativas com este e-mail. Espere alguns minutos e tente de novo.")
	}
	u, err := uc.d.Users.ByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && !uc.d.Hasher.Compare(u.PasswordHash, in.Password)) {
		uc.fail(email, now)
		return AuthOutput{}, domain.Unauthorized("E-mail ou senha incorretos.")
	}
	if err != nil {
		return AuthOutput{}, err
	}
	token, err := newSession(ctx, uc.d, u.ID)
	if err != nil {
		return AuthOutput{}, err
	}
	uc.mu.Lock()
	delete(uc.failures, email)
	uc.mu.Unlock()
	return AuthOutput{Token: token, User: *u}, nil
}

func (uc *Login) blocked(email string, now time.Time) bool {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	f, ok := uc.failures[email]
	if ok && now.Sub(f.since) >= loginWindow {
		delete(uc.failures, email)
		return false
	}
	return f.count >= loginMaxFailures
}

func (uc *Login) fail(email string, now time.Time) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	// E-mails inventados também entram: de tempos em tempos, tira os vencidos.
	if len(uc.failures) >= 10_000 {
		for k, f := range uc.failures {
			if now.Sub(f.since) >= loginWindow {
				delete(uc.failures, k)
			}
		}
	}
	f := uc.failures[email]
	if f.count == 0 {
		f.since = now
	}
	f.count++
	uc.failures[email] = f
}

// Authenticate troca um token pelo ID do usuário. O controller usa em toda requisição com token.
type Authenticate struct{ d *Deps }

func NewAuthenticate(d *Deps) *Authenticate { return &Authenticate{d} }

func (uc *Authenticate) Execute(ctx context.Context, token string) (string, error) {
	id, err := uc.d.Sessions.UserID(ctx, HashToken(token), uc.d.Clock.Now())
	if errors.Is(err, domain.ErrNotFound) {
		return "", domain.Unauthorized("Sua sessão expirou. Entre de novo.")
	}
	return id, err
}
