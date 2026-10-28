// Comando api sobe o servidor HTTP do Timeliner.
//
//	api                                              servidor HTTP
//	api create-user -handle h -name n -email e       cria uma conta (senha na 1a linha da entrada),
//	                                                 mesmo com SIGNUP=closed
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
	if len(os.Args) > 1 && os.Args[1] == "create-user" {
		if err := createUser(os.Args[2:]); err != nil {
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

	deps := newDeps(pool)
	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           controller.New(controller.UseCasesFrom(deps), log, pool.Ping),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	log.Info("Timeliner API no ar", "addr", addr, "signupClosed", deps.SignupClosed)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newDeps(pool *pgxpool.Pool) *usecase.Deps {
	deps := postgres.New(pool)
	deps.Hasher, deps.Clock, deps.IDs = platform.Bcrypt{}, platform.SystemClock{}, platform.RandomIDs{}
	// SIGNUP=closed: ninguém se cadastra pela API; contas só com create-user.
	deps.SignupClosed = os.Getenv("SIGNUP") == "closed"
	return deps
}

// createUser cria uma conta pelo terminal. A senha vem da entrada padrão, não de
// um argumento, para não ficar no histórico do shell nem na lista de processos.
func createUser(args []string) error {
	fs := flag.NewFlagSet("create-user", flag.ContinueOnError)
	handle := fs.String("handle", "", "nome de usuário (3 a 30: a-z, 0-9, _)")
	name := fs.String("name", "", "nome de exibição")
	email := fs.String("email", "", "e-mail de login")
	if err := fs.Parse(args); err != nil {
		return err
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return errors.New("senha não informada na entrada padrão")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, envOr("DATABASE_URL", "postgres://timeliner:timeliner@localhost:5432/timeliner?sslmode=disable"))
	if err != nil {
		return err
	}
	defer pool.Close()
	out, err := usecase.NewAdminRegister(newDeps(pool)).Execute(ctx, usecase.RegisterInput{
		Handle: *handle, Name: *name, Email: *email, Password: strings.TrimRight(line, "\r\n"),
	})
	if err != nil {
		var de *domain.Error
		if errors.As(err, &de) && len(de.Fields) > 0 {
			return fmt.Errorf("%s %v", de.Message, de.Fields)
		}
		return err
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
