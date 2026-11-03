package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

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
	// ClientIP é o IP de quem pede, para o limite de cadastros. Vazio não limita.
	ClientIP string
}

// Limite de cadastros por IP, contra criação de contas em massa. Folgado o
// bastante para uma sala de aula atrás do mesmo IP.
const (
	signupMaxPerIP = 10
	signupWindow   = time.Hour
)

// Register cria a conta e já devolve uma sessão.
type Register struct {
	d *Deps
	// ignoreClosed deixa o comando create-user criar contas com o cadastro fechado.
	ignoreClosed bool
	perIP        *limiter
}

func NewRegister(d *Deps) *Register {
	return &Register{d: d, perIP: newLimiter(signupMaxPerIP, signupWindow)}
}

// NewAdminRegister cria contas mesmo com o cadastro fechado (comando create-user).
func NewAdminRegister(d *Deps) *Register {
	return &Register{d: d, ignoreClosed: true, perIP: newLimiter(signupMaxPerIP, signupWindow)}
}

// Nomes que não podem virar @handle: confundem com o próprio site ou com a equipe.
var reservedHandles = map[string]bool{
	"me": true, "eu": true, "admin": true, "administrador": true, "adm": true, "root": true,
	"api": true, "www": true, "timeliner": true, "suporte": true, "support": true, "ajuda": true,
	"help": true, "contato": true, "equipe": true, "moderador": true, "moderacao": true,
	"sistema": true, "system": true, "oficial": true, "null": true, "undefined": true,
	"anonimo": true, "entrar": true, "cadastro": true, "cadastrar": true, "login": true,
}

// passwordProblem devolve o erro da senha nova, ou "" se ela serve. O bcrypt
// só lê 72 bytes; acima disso a senha seria cortada, então recusamos.
func passwordProblem(password, email, handle string) string {
	switch {
	case utf8.RuneCountInString(password) < 8:
		return "A senha precisa ter pelo menos 8 caracteres."
	case len(password) > 72:
		return "A senha pode ter no máximo 72 caracteres."
	case strings.TrimSpace(password) == "":
		return "A senha não pode ser só espaços."
	case strings.EqualFold(password, email) || (handle != "" && strings.EqualFold(password, handle)):
		return "A senha não pode ser igual ao e-mail ou ao nome de usuário."
	}
	return ""
}

func (uc *Register) Execute(ctx context.Context, in RegisterInput) (AuthOutput, error) {
	if uc.d.SignupClosed && !uc.ignoreClosed {
		return AuthOutput{}, domain.Forbidden("O cadastro está fechado no momento.")
	}
	in.Handle = strings.TrimSpace(in.Handle)
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	f := fieldErrors{}
	f.check(handleRe.MatchString(in.Handle), "handle", "Use de 3 a 30 letras minúsculas, números ou _.")
	f.check(!reservedHandles[in.Handle], "handle", "Esse nome é reservado. Escolha outro.")
	f.check(length(in.Name, 1, 80), "name", "O nome precisa ter de 1 a 80 caracteres.")
	f.check(validEmail(in.Email) && len(in.Email) <= 254, "email", "E-mail inválido.")
	if msg := passwordProblem(in.Password, in.Email, in.Handle); msg != "" {
		f.check(false, "password", msg)
	}
	if err := f.err(); err != nil {
		return AuthOutput{}, err
	}
	now := uc.d.Clock.Now()
	if uc.perIP.Blocked(in.ClientIP, now) {
		return AuthOutput{}, domain.TooMany("Muitas contas criadas a partir desta rede. Tente de novo mais tarde.")
	}
	uc.perIP.Hit(in.ClientIP, now)
	hash, err := uc.d.Hasher.Hash(in.Password)
	if err != nil {
		return AuthOutput{}, err
	}
	u := domain.User{
		ID: uc.d.IDs.NewID("usr"), Handle: in.Handle, Name: in.Name, Email: in.Email,
		PasswordHash: hash, Tint: domain.TintNeutral, CreatedAt: now,
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
	// Conta criada pelo create-user já nasce confirmada; as outras recebem o link.
	if uc.ignoreClosed {
		u.EmailVerifiedAt = &now
		if err := uc.d.Users.Update(ctx, &u); err != nil {
			return AuthOutput{}, err
		}
	} else if err := sendVerification(ctx, uc.d, u); err != nil {
		logError(ctx, uc.d, "falha ao mandar e-mail de confirmação", err)
	}
	return AuthOutput{Token: token, User: u}, nil
}

// SignupStatus diz ao site se mostra o cadastro.
type SignupStatus struct{ Open bool }

type GetSignupStatus struct{ d *Deps }

func NewGetSignupStatus(d *Deps) *GetSignupStatus { return &GetSignupStatus{d} }

func (uc *GetSignupStatus) Execute(context.Context, Empty) (SignupStatus, error) {
	return SignupStatus{Open: !uc.d.SignupClosed}, nil
}

type LoginInput struct {
	Email    string
	Password string
	// ClientIP é o IP de quem pede, para o limite por rede. Vazio não limita.
	ClientIP string
}

// Limites de senhas erradas: por e-mail, contra tentativa e erro numa conta; e
// por IP, contra quem testa muitas contas a partir do mesmo lugar.
const (
	loginMaxFailures      = 10
	loginMaxFailuresPerIP = 50
	loginWindow           = 15 * time.Minute
)

type Login struct {
	d       *Deps
	byEmail *limiter
	byIP    *limiter
}

func NewLogin(d *Deps) *Login {
	return &Login{d: d, byEmail: newLimiter(loginMaxFailures, loginWindow), byIP: newLimiter(loginMaxFailuresPerIP, loginWindow)}
}

func (uc *Login) Execute(ctx context.Context, in LoginInput) (AuthOutput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	now := uc.d.Clock.Now()
	if uc.byIP.Blocked(in.ClientIP, now) {
		return AuthOutput{}, domain.TooMany("Muitas tentativas de login a partir desta rede. Espere alguns minutos e tente de novo.")
	}
	if uc.byEmail.Blocked(email, now) {
		return AuthOutput{}, domain.Unauthorized("Muitas tentativas com este e-mail. Espere alguns minutos e tente de novo.")
	}
	u, err := uc.d.Users.ByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && !uc.d.Hasher.Compare(u.PasswordHash, in.Password)) {
		uc.byEmail.Hit(email, now)
		uc.byIP.Hit(in.ClientIP, now)
		return AuthOutput{}, domain.Unauthorized("E-mail ou senha incorretos.")
	}
	if err != nil {
		return AuthOutput{}, err
	}
	token, err := newSession(ctx, uc.d, u.ID)
	if err != nil {
		return AuthOutput{}, err
	}
	uc.byEmail.Reset(email)
	return AuthOutput{Token: token, User: *u}, nil
}

// Logout apaga a sessão do token: ele deixa de valer na hora, não só no navegador.
type Logout struct{ d *Deps }

func NewLogout(d *Deps) *Logout { return &Logout{d} }

func (uc *Logout) Execute(ctx context.Context, token string) (Empty, error) {
	if token == "" {
		return Empty{}, nil
	}
	return Empty{}, uc.d.Sessions.Delete(ctx, HashToken(token))
}

type ChangePasswordInput struct {
	ActorID string
	// Token é a sessão atual, que continua valendo; as outras são encerradas.
	Token           string
	CurrentPassword string
	NewPassword     string
}

// ChangePassword troca a senha de quem está logado, conferindo a atual, e
// encerra as outras sessões da conta.
type ChangePassword struct {
	d        *Deps
	failures *limiter
}

func NewChangePassword(d *Deps) *ChangePassword {
	return &ChangePassword{d: d, failures: newLimiter(loginMaxFailures, loginWindow)}
}

func (uc *ChangePassword) Execute(ctx context.Context, in ChangePasswordInput) (Empty, error) {
	if err := requireActor(in.ActorID); err != nil {
		return Empty{}, err
	}
	now := uc.d.Clock.Now()
	if uc.failures.Blocked(in.ActorID, now) {
		return Empty{}, domain.TooMany("Muitas tentativas com a senha errada. Espere alguns minutos e tente de novo.")
	}
	u, err := uc.d.Users.ByID(ctx, in.ActorID)
	if err != nil {
		return Empty{}, notFound(err, "Usuário não encontrado.")
	}
	if !uc.d.Hasher.Compare(u.PasswordHash, in.CurrentPassword) {
		uc.failures.Hit(in.ActorID, now)
		return Empty{}, domain.Invalid(map[string]string{"currentPassword": "Senha atual incorreta."})
	}
	if msg := passwordProblem(in.NewPassword, u.Email, u.Handle); msg != "" {
		return Empty{}, domain.Invalid(map[string]string{"newPassword": msg})
	}
	uc.failures.Reset(in.ActorID)
	return Empty{}, setPassword(ctx, uc.d, u, in.NewPassword, HashToken(in.Token))
}

func setPassword(ctx context.Context, d *Deps, u *domain.User, password, keepSession string) error {
	hash, err := d.Hasher.Hash(password)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := d.Users.Update(ctx, u); err != nil {
			return err
		}
		return d.Sessions.DeleteByUser(ctx, u.ID, keepSession)
	})
}

type ResetPasswordInput struct {
	Email    string
	Password string
}

// ResetPassword define uma senha nova sem a antiga e encerra todas as sessões.
// Só o comando reset-password do servidor usa: não tem rota na API.
type ResetPassword struct{ d *Deps }

func NewResetPassword(d *Deps) *ResetPassword { return &ResetPassword{d} }

func (uc *ResetPassword) Execute(ctx context.Context, in ResetPasswordInput) (domain.User, error) {
	u, err := uc.d.Users.ByEmail(ctx, strings.ToLower(strings.TrimSpace(in.Email)))
	if err != nil {
		return domain.User{}, notFound(err, "Nenhuma conta com esse e-mail.")
	}
	if msg := passwordProblem(in.Password, u.Email, u.Handle); msg != "" {
		return domain.User{}, domain.Invalid(map[string]string{"password": msg})
	}
	if err := setPassword(ctx, uc.d, u, in.Password, ""); err != nil {
		return domain.User{}, err
	}
	return *u, nil
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
