package usecase_test

import (
	"testing"
	"time"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func TestRegisterLoginAuthenticate(t *testing.T) {
	d := newDeps()
	reg := must[usecase.AuthOutput](t)(usecase.NewRegister(d).Execute(ctx, usecase.RegisterInput{
		Handle: "ana", Name: " Ana ", Email: "ANA@exemplo.com", Password: "12345678",
	}))
	if reg.User.Name != "Ana" || reg.User.Email != "ana@exemplo.com" || reg.User.PasswordHash == "12345678" {
		t.Fatalf("usuário = %+v", reg.User)
	}
	id := must[string](t)(usecase.NewAuthenticate(d).Execute(ctx, reg.Token))
	if id != reg.User.ID {
		t.Fatalf("authenticate = %q", id)
	}

	login := must[usecase.AuthOutput](t)(usecase.NewLogin(d).Execute(ctx, usecase.LoginInput{Email: "ana@exemplo.com", Password: "12345678"}))
	if login.Token == reg.Token {
		t.Fatal("login devia criar outra sessão")
	}
	_, err := usecase.NewLogin(d).Execute(ctx, usecase.LoginInput{Email: "ana@exemplo.com", Password: "errada"})
	wantKind(t, err, domain.KindUnauthorized)
	_, err = usecase.NewLogin(d).Execute(ctx, usecase.LoginInput{Email: "ninguem@exemplo.com", Password: "12345678"})
	wantKind(t, err, domain.KindUnauthorized)
	_, err = usecase.NewAuthenticate(d).Execute(ctx, "token-falso")
	wantKind(t, err, domain.KindUnauthorized)
}

func TestRegisterValidationAndConflict(t *testing.T) {
	d := newDeps()
	_, err := usecase.NewRegister(d).Execute(ctx, usecase.RegisterInput{Handle: "A!", Email: "x", Password: "1"})
	wantKind(t, err, domain.KindValidation)
	fields := err.(*domain.Error).Fields
	for _, f := range []string{"handle", "name", "email", "password"} {
		if fields[f] == "" {
			t.Errorf("faltou erro em %s: %v", f, fields)
		}
	}

	register(t, d, "ana")
	_, err = usecase.NewRegister(d).Execute(ctx, usecase.RegisterInput{Handle: "ana", Name: "Outra", Email: "outra@exemplo.com", Password: "12345678"})
	wantKind(t, err, domain.KindConflict)
	if err.(*domain.Error).Fields["handle"] == "" {
		t.Fatalf("conflito sem campo: %v", err)
	}
}

func TestRegisterClosed(t *testing.T) {
	d := newDeps()
	d.SignupClosed = true
	_, err := usecase.NewRegister(d).Execute(ctx, usecase.RegisterInput{Handle: "ana", Name: "Ana", Email: "ana@exemplo.com", Password: "12345678"})
	wantKind(t, err, domain.KindForbidden)

	out := must[usecase.AuthOutput](t)(usecase.NewAdminRegister(d).Execute(ctx, usecase.RegisterInput{Handle: "ana", Name: "Ana", Email: "ana@exemplo.com", Password: "12345678"}))
	if out.User.Handle != "ana" {
		t.Fatalf("usuário = %+v", out.User)
	}
}

func TestLoginBlocksAfterTooManyFailures(t *testing.T) {
	d := newDeps()
	register(t, d, "ana")
	login := usecase.NewLogin(d)
	for i := 0; i < 10; i++ {
		_, err := login.Execute(ctx, usecase.LoginInput{Email: "ana@exemplo.com", Password: "errada"})
		wantKind(t, err, domain.KindUnauthorized)
	}
	// Bloqueado: nem a senha certa entra até a janela passar.
	_, err := login.Execute(ctx, usecase.LoginInput{Email: "ANA@exemplo.com", Password: "senha-forte"})
	wantKind(t, err, domain.KindUnauthorized)
	if msg := err.(*domain.Error).Message; msg == "E-mail ou senha incorretos." {
		t.Fatalf("esperava aviso de bloqueio, veio %q", msg)
	}

	d.Clock.(*fakeClock).t = d.Clock.Now().Add(15 * time.Minute)
	must[usecase.AuthOutput](t)(login.Execute(ctx, usecase.LoginInput{Email: "ana@exemplo.com", Password: "senha-forte"}))
}
