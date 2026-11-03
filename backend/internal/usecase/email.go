package usecase

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

// Validade dos links mandados por e-mail.
const (
	verifyEmailTTL   = 7 * 24 * time.Hour
	resetPasswordTTL = time.Hour
)

// Limites de pedidos de e-mail, para ninguém usar o site para encher a caixa
// de alguém (nem gastar a cota do provedor).
const (
	mailMaxPerAddress = 3
	mailMaxPerIP      = 20
	mailWindow        = time.Hour
)

// newAuthToken cria um link de uso único e devolve o token em claro (vai só no e-mail).
func newAuthToken(ctx context.Context, d *Deps, userID string, purpose domain.TokenPurpose, ttl time.Duration) (string, error) {
	token := d.IDs.NewToken()
	now := d.Clock.Now()
	err := d.AuthTokens.Create(ctx, domain.AuthToken{
		TokenHash: HashToken(token), UserID: userID, Purpose: purpose, CreatedAt: now, ExpiresAt: now.Add(ttl),
	})
	return token, err
}

func appLink(d *Deps, path, token string) string {
	return strings.TrimRight(d.AppURL, "/") + path + "?token=" + token
}

// mailLink monta um e-mail simples com um botão para o link.
func mailLink(to, subject, greeting, intro, button, link, outro string) domain.Email {
	text := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n— Timeliner\n", greeting, intro, link, outro)
	h := fmt.Sprintf(`<div style="font-family:system-ui,sans-serif;max-width:480px;margin:auto;color:#1c1b19;line-height:1.5">
<p>%s</p><p>%s</p>
<p style="margin:28px 0"><a href="%s" style="background:#1c1b19;color:#fff;padding:12px 20px;border-radius:8px;text-decoration:none;display:inline-block">%s</a></p>
<p style="font-size:13px;color:#6b6760">Se o botão não abrir, copie este endereço no navegador:<br><span style="word-break:break-all">%s</span></p>
<p style="font-size:13px;color:#6b6760">%s</p><p>— Timeliner</p></div>`,
		html.EscapeString(greeting), html.EscapeString(intro), html.EscapeString(link), html.EscapeString(button),
		html.EscapeString(link), html.EscapeString(outro))
	return domain.Email{To: to, Subject: subject, Text: text, HTML: h}
}

func logError(ctx context.Context, d *Deps, msg string, err error) {
	if d.Log != nil {
		d.Log.ErrorContext(ctx, msg, "err", err)
	}
}

func sendMail(ctx context.Context, d *Deps, e domain.Email) error {
	if d.Mailer == nil {
		return nil
	}
	return d.Mailer.Send(ctx, e)
}

// sendVerification manda o link de confirmação. Falhar aqui não desfaz o cadastro:
// a pessoa pode pedir outro link depois. Os e-mails não levam o nome digitado no
// cadastro: quem usa o e-mail de outra pessoa não consegue pôr texto seu na caixa dela.
func sendVerification(ctx context.Context, d *Deps, u domain.User) error {
	if err := d.AuthTokens.DeleteByUser(ctx, u.ID, domain.TokenVerifyEmail); err != nil {
		return err
	}
	token, err := newAuthToken(ctx, d, u.ID, domain.TokenVerifyEmail, verifyEmailTTL)
	if err != nil {
		return err
	}
	return sendMail(ctx, d, mailLink(u.Email, "Confirme seu e-mail no Timeliner",
		"Olá!",
		"Falta só confirmar que este e-mail é seu. É por ele que você recupera a senha se esquecer.",
		"Confirmar e-mail", appLink(d, "/confirmar-email", token),
		"O link vale por 7 dias. Se não foi você que criou a conta, pode ignorar este e-mail."))
}

// VerifyEmail confirma o e-mail pelo link mandado no cadastro.
type VerifyEmail struct{ d *Deps }

func NewVerifyEmail(d *Deps) *VerifyEmail { return &VerifyEmail{d} }

func (uc *VerifyEmail) Execute(ctx context.Context, token string) (domain.User, error) {
	var out domain.User
	err := uc.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		id, err := uc.d.AuthTokens.Consume(ctx, HashToken(token), domain.TokenVerifyEmail, uc.d.Clock.Now())
		if errors.Is(err, domain.ErrNotFound) {
			return &domain.Error{Kind: domain.KindValidation, Message: "Este link de confirmação já foi usado ou venceu. Peça outro no seu perfil."}
		}
		if err != nil {
			return err
		}
		u, err := uc.d.Users.ByID(ctx, id)
		if err != nil {
			return notFound(err, "Usuário não encontrado.")
		}
		if u.EmailVerifiedAt == nil {
			now := uc.d.Clock.Now()
			u.EmailVerifiedAt = &now
			if err := uc.d.Users.Update(ctx, u); err != nil {
				return err
			}
		}
		out = *u
		return nil
	})
	return out, err
}

// ResendVerification manda de novo o link de confirmação para quem está logado.
type ResendVerification struct {
	d      *Deps
	perUse *limiter
}

func NewResendVerification(d *Deps) *ResendVerification {
	return &ResendVerification{d: d, perUse: newLimiter(mailMaxPerAddress, mailWindow)}
}

func (uc *ResendVerification) Execute(ctx context.Context, in ActorInput) (Empty, error) {
	if err := requireActor(in.ActorID); err != nil {
		return Empty{}, err
	}
	u, err := uc.d.Users.ByID(ctx, in.ActorID)
	if err != nil {
		return Empty{}, notFound(err, "Usuário não encontrado.")
	}
	if u.EmailVerifiedAt != nil {
		return Empty{}, nil
	}
	now := uc.d.Clock.Now()
	if uc.perUse.Blocked(u.ID, now) {
		return Empty{}, domain.TooMany("Já mandamos alguns e-mails há pouco. Confira a caixa de spam ou espere um pouco.")
	}
	uc.perUse.Hit(u.ID, now)
	return Empty{}, sendVerification(ctx, uc.d, *u)
}

type RequestPasswordResetInput struct {
	Email    string
	ClientIP string
}

// RequestPasswordReset manda o link de "esqueci minha senha". Responde igual
// exista a conta ou não, para não revelar quem tem conta.
type RequestPasswordReset struct {
	d       *Deps
	byEmail *limiter
	byIP    *limiter
}

func NewRequestPasswordReset(d *Deps) *RequestPasswordReset {
	return &RequestPasswordReset{d: d, byEmail: newLimiter(mailMaxPerAddress, mailWindow), byIP: newLimiter(mailMaxPerIP, mailWindow)}
}

func (uc *RequestPasswordReset) Execute(ctx context.Context, in RequestPasswordResetInput) (Empty, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if !validEmail(email) {
		return Empty{}, domain.Invalid(map[string]string{"email": "E-mail inválido."})
	}
	now := uc.d.Clock.Now()
	if uc.byIP.Blocked(in.ClientIP, now) {
		return Empty{}, domain.TooMany("Muitos pedidos a partir desta rede. Tente de novo mais tarde.")
	}
	uc.byIP.Hit(in.ClientIP, now)
	// Passou do limite deste e-mail: finge que mandou, como para um e-mail sem conta.
	if uc.byEmail.Blocked(email, now) {
		return Empty{}, nil
	}
	uc.byEmail.Hit(email, now)
	u, err := uc.d.Users.ByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return Empty{}, nil
	}
	if err != nil {
		return Empty{}, err
	}
	if err := uc.d.AuthTokens.DeleteByUser(ctx, u.ID, domain.TokenResetPassword); err != nil {
		return Empty{}, err
	}
	token, err := newAuthToken(ctx, uc.d, u.ID, domain.TokenResetPassword, resetPasswordTTL)
	if err != nil {
		return Empty{}, err
	}
	err = sendMail(ctx, uc.d, mailLink(u.Email, "Redefinir sua senha no Timeliner",
		"Olá!",
		"Recebemos um pedido para redefinir a senha da sua conta (@"+u.Handle+").",
		"Criar senha nova", appLink(uc.d, "/redefinir-senha", token),
		"O link vale por 1 hora e só funciona uma vez. Se não foi você que pediu, ignore este e-mail: sua senha continua a mesma."))
	if err != nil {
		// A resposta não muda (não revela se a conta existe); o erro fica no log.
		logError(ctx, uc.d, "falha ao mandar e-mail de senha", err)
	}
	return Empty{}, nil
}

type ConfirmPasswordResetInput struct {
	Token    string
	Password string
}

// ConfirmPasswordReset troca a senha pelo link do e-mail, encerra as outras
// sessões e já entra na conta. Abrir o link prova o e-mail, então ele fica confirmado.
type ConfirmPasswordReset struct{ d *Deps }

func NewConfirmPasswordReset(d *Deps) *ConfirmPasswordReset { return &ConfirmPasswordReset{d} }

func (uc *ConfirmPasswordReset) Execute(ctx context.Context, in ConfirmPasswordResetInput) (AuthOutput, error) {
	var out AuthOutput
	err := uc.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		id, err := uc.d.AuthTokens.Consume(ctx, HashToken(in.Token), domain.TokenResetPassword, uc.d.Clock.Now())
		if errors.Is(err, domain.ErrNotFound) {
			return &domain.Error{Kind: domain.KindValidation, Message: "Este link já foi usado ou venceu. Peça outro em \"Esqueci minha senha\"."}
		}
		if err != nil {
			return err
		}
		u, err := uc.d.Users.ByID(ctx, id)
		if err != nil {
			return notFound(err, "Usuário não encontrado.")
		}
		if msg := passwordProblem(in.Password, u.Email, u.Handle); msg != "" {
			return domain.Invalid(map[string]string{"password": msg})
		}
		if u.EmailVerifiedAt == nil {
			now := uc.d.Clock.Now()
			u.EmailVerifiedAt = &now
		}
		if err := setPassword(ctx, uc.d, u, in.Password, ""); err != nil {
			return err
		}
		token, err := newSession(ctx, uc.d, u.ID)
		out = AuthOutput{Token: token, User: *u}
		return err
	})
	return out, err
}
