package usecase_test

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

// inbox guarda os e-mails "mandados" nos testes.
type inbox struct{ sent []domain.Email }

func (b *inbox) Send(_ context.Context, e domain.Email) error {
	b.sent = append(b.sent, e)
	return nil
}

var tokenRe = regexp.MustCompile(`\?token=(\S+)`)

// last devolve o token do último e-mail para to.
func (b *inbox) last(t *testing.T, to, path string) string {
	t.Helper()
	for i := len(b.sent) - 1; i >= 0; i-- {
		e := b.sent[i]
		if e.To == to && strings.Contains(e.Text, path) {
			return tokenRe.FindStringSubmatch(e.Text)[1]
		}
	}
	t.Fatalf("nenhum e-mail para %s com %s: %+v", to, path, b.sent)
	return ""
}

func withInbox() (*usecase.Deps, *inbox) {
	d := newDeps()
	b := &inbox{}
	d.Mailer, d.AppURL = b, "https://timeliner.test/"
	return d, b
}

func TestRegisterRules(t *testing.T) {
	d := newDeps()
	cases := map[string]usecase.RegisterInput{
		"handle":   {Handle: "admin", Name: "A", Email: "a@exemplo.com", Password: "12345678"},
		"password": {Handle: "ana", Name: "A", Email: "a@exemplo.com", Password: strings.Repeat("x", 73)},
	}
	for field, in := range cases {
		_, err := usecase.NewRegister(d).Execute(ctx, in)
		wantKind(t, err, domain.KindValidation)
		if err.(*domain.Error).Fields[field] == "" {
			t.Errorf("%s: faltou erro no campo: %v", field, err)
		}
	}
	_, err := usecase.NewRegister(d).Execute(ctx, usecase.RegisterInput{Handle: "ana_maria", Name: "A", Email: "ana_maria@x.com", Password: "ANA_MARIA"})
	wantKind(t, err, domain.KindValidation)
}

func TestRegisterLimitPerIP(t *testing.T) {
	d := newDeps()
	reg := usecase.NewRegister(d)
	for i := 0; i < 10; i++ {
		h := "user" + string(rune('a'+i))
		must[usecase.AuthOutput](t)(reg.Execute(ctx, usecase.RegisterInput{Handle: h, Name: h, Email: h + "@x.com", Password: "senha-forte", ClientIP: "1.2.3.4"}))
	}
	_, err := reg.Execute(ctx, usecase.RegisterInput{Handle: "outro", Name: "o", Email: "o@x.com", Password: "senha-forte", ClientIP: "1.2.3.4"})
	wantKind(t, err, domain.KindTooMany)
	// Outra rede segue podendo.
	must[usecase.AuthOutput](t)(reg.Execute(ctx, usecase.RegisterInput{Handle: "outro", Name: "o", Email: "o@x.com", Password: "senha-forte", ClientIP: "5.6.7.8"}))
	// Passada a janela, a mesma rede volta a poder.
	d.Clock.(*fakeClock).t = d.Clock.Now().Add(time.Hour)
	must[usecase.AuthOutput](t)(reg.Execute(ctx, usecase.RegisterInput{Handle: "mais_um", Name: "m", Email: "m@x.com", Password: "senha-forte", ClientIP: "1.2.3.4"}))
}

func TestLoginLimitPerIP(t *testing.T) {
	d := newDeps()
	login := usecase.NewLogin(d)
	for i := 0; i < 50; i++ {
		_, err := login.Execute(ctx, usecase.LoginInput{Email: "x" + string(rune('a'+i%26)) + "@x.com", Password: "errada", ClientIP: "9.9.9.9"})
		wantKind(t, err, domain.KindUnauthorized)
	}
	_, err := login.Execute(ctx, usecase.LoginInput{Email: "nova@x.com", Password: "errada", ClientIP: "9.9.9.9"})
	wantKind(t, err, domain.KindTooMany)
}

func TestLogoutEndsSession(t *testing.T) {
	d := newDeps()
	out := must[usecase.AuthOutput](t)(usecase.NewRegister(d).Execute(ctx, usecase.RegisterInput{Handle: "ana", Name: "Ana", Email: "ana@x.com", Password: "senha-forte"}))
	must[usecase.Empty](t)(usecase.NewLogout(d).Execute(ctx, out.Token))
	_, err := usecase.NewAuthenticate(d).Execute(ctx, out.Token)
	wantKind(t, err, domain.KindUnauthorized)
}

func TestChangePassword(t *testing.T) {
	d := newDeps()
	ana := register(t, d, "ana")
	here := must[usecase.AuthOutput](t)(usecase.NewLogin(d).Execute(ctx, usecase.LoginInput{Email: "ana@exemplo.com", Password: "senha-forte"}))
	other := must[usecase.AuthOutput](t)(usecase.NewLogin(d).Execute(ctx, usecase.LoginInput{Email: "ana@exemplo.com", Password: "senha-forte"}))
	cp := usecase.NewChangePassword(d)

	_, err := cp.Execute(ctx, usecase.ChangePasswordInput{ActorID: ana.ID, Token: here.Token, CurrentPassword: "errada", NewPassword: "nova-senha-1"})
	wantKind(t, err, domain.KindValidation)
	if err.(*domain.Error).Fields["currentPassword"] == "" {
		t.Fatalf("erro sem campo: %v", err)
	}
	_, err = cp.Execute(ctx, usecase.ChangePasswordInput{ActorID: ana.ID, Token: here.Token, CurrentPassword: "senha-forte", NewPassword: "curta"})
	wantKind(t, err, domain.KindValidation)

	must[usecase.Empty](t)(cp.Execute(ctx, usecase.ChangePasswordInput{ActorID: ana.ID, Token: here.Token, CurrentPassword: "senha-forte", NewPassword: "nova-senha-1"}))
	must[string](t)(usecase.NewAuthenticate(d).Execute(ctx, here.Token))
	_, err = usecase.NewAuthenticate(d).Execute(ctx, other.Token)
	wantKind(t, err, domain.KindUnauthorized)
	must[usecase.AuthOutput](t)(usecase.NewLogin(d).Execute(ctx, usecase.LoginInput{Email: "ana@exemplo.com", Password: "nova-senha-1"}))
}

func TestVerifyEmail(t *testing.T) {
	d, box := withInbox()
	out := must[usecase.AuthOutput](t)(usecase.NewRegister(d).Execute(ctx, usecase.RegisterInput{Handle: "ana", Name: "Ana", Email: "ana@x.com", Password: "senha-forte"}))
	if out.User.EmailVerifiedAt != nil {
		t.Fatal("conta nova já veio confirmada")
	}
	if len(box.sent) != 1 || !strings.Contains(box.sent[0].Text, "https://timeliner.test/confirmar-email?token=") {
		t.Fatalf("e-mails = %+v", box.sent)
	}
	token := box.last(t, "ana@x.com", "/confirmar-email")

	// Pedir de novo invalida o link anterior.
	must[usecase.Empty](t)(usecase.NewResendVerification(d).Execute(ctx, usecase.ActorInput{ActorID: out.User.ID}))
	_, err := usecase.NewVerifyEmail(d).Execute(ctx, token)
	wantKind(t, err, domain.KindValidation)

	u := must[domain.User](t)(usecase.NewVerifyEmail(d).Execute(ctx, box.last(t, "ana@x.com", "/confirmar-email")))
	if u.EmailVerifiedAt == nil {
		t.Fatal("e-mail não ficou confirmado")
	}
	me := must[domain.User](t)(usecase.NewGetMe(d).Execute(ctx, usecase.ActorInput{ActorID: out.User.ID}))
	if me.EmailVerifiedAt == nil {
		t.Fatal("confirmação não foi salva")
	}
	// Já confirmado: reenviar não manda nada.
	n := len(box.sent)
	must[usecase.Empty](t)(usecase.NewResendVerification(d).Execute(ctx, usecase.ActorInput{ActorID: out.User.ID}))
	if len(box.sent) != n {
		t.Fatal("mandou e-mail para quem já confirmou")
	}
}

func TestAdminRegisterIsVerified(t *testing.T) {
	d, box := withInbox()
	out := must[usecase.AuthOutput](t)(usecase.NewAdminRegister(d).Execute(ctx, usecase.RegisterInput{Handle: "ana", Name: "Ana", Email: "ana@x.com", Password: "senha-forte"}))
	if out.User.EmailVerifiedAt == nil || len(box.sent) != 0 {
		t.Fatalf("usuário = %+v, e-mails = %d", out.User, len(box.sent))
	}
}

func TestPasswordResetByEmail(t *testing.T) {
	d, box := withInbox()
	old := must[usecase.AuthOutput](t)(usecase.NewRegister(d).Execute(ctx, usecase.RegisterInput{Handle: "ana", Name: "Ana", Email: "ana@x.com", Password: "senha-forte"}))
	req := usecase.NewRequestPasswordReset(d)

	// E-mail sem conta: responde igual e não manda nada.
	n := len(box.sent)
	must[usecase.Empty](t)(req.Execute(ctx, usecase.RequestPasswordResetInput{Email: "ninguem@x.com"}))
	if len(box.sent) != n {
		t.Fatal("mandou e-mail para quem não tem conta")
	}

	must[usecase.Empty](t)(req.Execute(ctx, usecase.RequestPasswordResetInput{Email: " ANA@x.com "}))
	token := box.last(t, "ana@x.com", "/redefinir-senha")

	confirm := usecase.NewConfirmPasswordReset(d)
	_, err := confirm.Execute(ctx, usecase.ConfirmPasswordResetInput{Token: "falso", Password: "nova-senha-1"})
	wantKind(t, err, domain.KindValidation)

	out := must[usecase.AuthOutput](t)(confirm.Execute(ctx, usecase.ConfirmPasswordResetInput{Token: token, Password: "nova-senha-1"}))
	if out.Token == "" || out.User.EmailVerifiedAt == nil {
		t.Fatalf("saída = %+v", out)
	}
	_, err = usecase.NewAuthenticate(d).Execute(ctx, old.Token)
	wantKind(t, err, domain.KindUnauthorized)
	must[usecase.AuthOutput](t)(usecase.NewLogin(d).Execute(ctx, usecase.LoginInput{Email: "ana@x.com", Password: "nova-senha-1"}))

	// O link só vale uma vez.
	_, err = confirm.Execute(ctx, usecase.ConfirmPasswordResetInput{Token: token, Password: "outra-senha-2"})
	wantKind(t, err, domain.KindValidation)
}

func TestPasswordResetExpiresAndLimits(t *testing.T) {
	d, box := withInbox()
	register(t, d, "ana")
	req := usecase.NewRequestPasswordReset(d)
	must[usecase.Empty](t)(req.Execute(ctx, usecase.RequestPasswordResetInput{Email: "ana@exemplo.com"}))
	token := box.last(t, "ana@exemplo.com", "/redefinir-senha")
	d.Clock.(*fakeClock).t = d.Clock.Now().Add(2 * time.Hour)
	_, err := usecase.NewConfirmPasswordReset(d).Execute(ctx, usecase.ConfirmPasswordResetInput{Token: token, Password: "nova-senha-1"})
	wantKind(t, err, domain.KindValidation)

	// Mais de 3 pedidos na hora para o mesmo e-mail: não manda mais, mas responde igual.
	for i := 0; i < 5; i++ {
		must[usecase.Empty](t)(req.Execute(ctx, usecase.RequestPasswordResetInput{Email: "ana@exemplo.com"}))
	}
	resets := 0
	for _, e := range box.sent {
		if strings.Contains(e.Text, "/redefinir-senha") {
			resets++
		}
	}
	if resets != 4 { // 1 do começo + 3 depois que a janela passou
		t.Fatalf("mandou %d e-mails de senha", resets)
	}
}

func TestResetPasswordCommand(t *testing.T) {
	d := newDeps()
	register(t, d, "ana")
	s := must[usecase.AuthOutput](t)(usecase.NewLogin(d).Execute(ctx, usecase.LoginInput{Email: "ana@exemplo.com", Password: "senha-forte"}))
	must[domain.User](t)(usecase.NewResetPassword(d).Execute(ctx, usecase.ResetPasswordInput{Email: "ana@exemplo.com", Password: "nova-senha-1"}))
	_, err := usecase.NewAuthenticate(d).Execute(ctx, s.Token)
	wantKind(t, err, domain.KindUnauthorized)
	_, err = usecase.NewResetPassword(d).Execute(ctx, usecase.ResetPasswordInput{Email: "ninguem@x.com", Password: "nova-senha-1"})
	wantKind(t, err, domain.KindNotFound)
}

// Quem cadastra com o e-mail de outra pessoa não põe texto seu na caixa dela.
func TestMailsDoNotCarryTypedName(t *testing.T) {
	d, box := withInbox()
	spam := "Ganhe um premio em golpe.example"
	must[usecase.AuthOutput](t)(usecase.NewRegister(d).Execute(ctx, usecase.RegisterInput{Handle: "ana", Name: spam, Email: "ana@x.com", Password: "senha-forte"}))
	must[usecase.Empty](t)(usecase.NewRequestPasswordReset(d).Execute(ctx, usecase.RequestPasswordResetInput{Email: "ana@x.com"}))
	if len(box.sent) != 2 {
		t.Fatalf("e-mails = %d", len(box.sent))
	}
	for _, m := range box.sent {
		if strings.Contains(m.Text, spam) || strings.Contains(m.HTML, spam) || strings.Contains(m.Subject, spam) {
			t.Fatalf("e-mail com o nome digitado: %+v", m)
		}
	}
}
