// Comando api sobe o servidor HTTP do Timeliner.
//
//	api                                              servidor HTTP
//	api create-user -handle h -name n -email e       cria uma conta (senha na 1a linha da entrada),
//	                                                 mesmo com SIGNUP=closed
//	api reset-password -email e                      define uma senha nova (1a linha da entrada)
//	                                                 e encerra as sessões da conta
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klausner17/timeliner/backend/internal/controller"
	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/platform"
	"github.com/klausner17/timeliner/backend/internal/repository/postgres"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if len(os.Args) > 1 && (os.Args[1] == "create-user" || os.Args[1] == "reset-password") {
		cmd := createUser
		if os.Args[1] == "reset-password" {
			cmd = resetPassword
		}
		if err := cmd(os.Args[2:], log); err != nil {
			fmt.Fprintln(os.Stderr, "erro:", err)
			os.Exit(1)
		}
		return
	}
	if err := run(log); err != nil {
		log.Error("servidor parou", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, envOr("DATABASE_URL", "postgres://timeliner:timeliner@localhost:5432/timeliner?sslmode=disable"))
	if err != nil {
		return err
	}
	defer pool.Close()
	if os.Getenv("MIGRATE") != "false" {
		if err := postgres.Migrate(ctx, pool); err != nil {
			return err
		}
	}

	deps := newDeps(pool, log)
	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr: addr,
		Handler: controller.NewWithConfig(controller.UseCasesFrom(deps), log, pool.Ping, controller.Config{
			ClientIPHeader: clientIPHeader(),
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	_, resend := deps.Mailer.(platform.Resend)
	log.Info("Timeliner API no ar", "addr", addr, "signupClosed", deps.SignupClosed, "emailViaResend", resend, "appURL", deps.AppURL)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newDeps(pool *pgxpool.Pool, log *slog.Logger) *usecase.Deps {
	deps := postgres.New(pool)
	deps.Log = log
	deps.Hasher, deps.Clock, deps.IDs = platform.Bcrypt{}, platform.SystemClock{}, platform.RandomIDs{}
	// Prévia de links só dos sites conhecidos; LINK_PREVIEW_EXTRA_DOMAINS soma outros.
	deps.Previews = platform.NewLinkPreviews(platform.PreviewDomains(os.Getenv("LINK_PREVIEW_EXTRA_DOMAINS")), false)
	// SIGNUP=closed: ninguém se cadastra pela API; contas só com create-user.
	deps.SignupClosed = os.Getenv("SIGNUP") == "closed"
	// E-mails (confirmação e senha) pelo Resend; sem chave, os links vão para o log.
	deps.AppURL = envOr("APP_URL", "http://localhost:8000")
	if key := os.Getenv("RESEND_API_KEY"); key != "" {
		deps.Mailer = platform.Resend{APIKey: key, From: envOr("EMAIL_FROM", "Timeliner <onboarding@resend.dev>")}
	} else {
		deps.Mailer = platform.LogMailer{Log: log}
	}
	return deps
}

// clientIPHeader é o cabeçalho com o IP do cliente posto pelo proxy (padrão
// X-Real-IP, do nginx do frontend). CLIENT_IP_HEADER=none usa o endereço da conexão.
func clientIPHeader() string {
	h := envOr("CLIENT_IP_HEADER", "X-Real-IP")
	if h == "none" {
		return ""
	}
	return h
}

// readPassword lê a senha da primeira linha da entrada padrão.
func readPassword() (string, error) {
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", errors.New("senha não informada na entrada padrão")
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func commandError(err error) error {
	var de *domain.Error
	if errors.As(err, &de) && len(de.Fields) > 0 {
		return fmt.Errorf("%s %v", de.Message, de.Fields)
	}
	return err
}

// resetPassword define a senha de uma conta pelo terminal, para quem esqueceu
// quando o e-mail não está configurado.
func resetPassword(args []string, log *slog.Logger) error {
	fs := flag.NewFlagSet("reset-password", flag.ContinueOnError)
	email := fs.String("email", "", "e-mail da conta")
	if err := fs.Parse(args); err != nil {
		return err
	}
	password, err := readPassword()
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, envOr("DATABASE_URL", "postgres://timeliner:timeliner@localhost:5432/timeliner?sslmode=disable"))
	if err != nil {
		return err
	}
	defer pool.Close()
	u, err := usecase.NewResetPassword(newDeps(pool, log)).Execute(ctx, usecase.ResetPasswordInput{Email: *email, Password: password})
	if err != nil {
		return commandError(err)
	}
	fmt.Printf("senha trocada: @%s <%s>\n", u.Handle, u.Email)
	return nil
}

// createUser cria uma conta pelo terminal. A senha vem da entrada padrão, não de
// um argumento, para não ficar no histórico do shell nem na lista de processos.
func createUser(args []string, log *slog.Logger) error {
	fs := flag.NewFlagSet("create-user", flag.ContinueOnError)
	handle := fs.String("handle", "", "nome de usuário (3 a 30: a-z, 0-9, _)")
	name := fs.String("name", "", "nome de exibição")
	email := fs.String("email", "", "e-mail de login")
	if err := fs.Parse(args); err != nil {
		return err
	}
	password, err := readPassword()
	if err != nil {
		return err
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, envOr("DATABASE_URL", "postgres://timeliner:timeliner@localhost:5432/timeliner?sslmode=disable"))
	if err != nil {
		return err
	}
	defer pool.Close()
	out, err := usecase.NewAdminRegister(newDeps(pool, log)).Execute(ctx, usecase.RegisterInput{
		Handle: *handle, Name: *name, Email: *email, Password: password,
	})
	if err != nil {
		return commandError(err)
	}
	fmt.Printf("conta criada: @%s <%s>\n", out.User.Handle, out.User.Email)
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
