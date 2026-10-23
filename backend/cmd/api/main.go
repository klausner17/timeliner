// Comando api sobe o servidor HTTP do Timeliner.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klausner17/timeliner/backend/internal/controller"
	"github.com/klausner17/timeliner/backend/internal/platform"
	"github.com/klausner17/timeliner/backend/internal/repository/postgres"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
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

	deps := postgres.New(pool)
	deps.Hasher, deps.Clock, deps.IDs = platform.Bcrypt{}, platform.SystemClock{}, platform.RandomIDs{}

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
	log.Info("Timeliner API no ar", "addr", addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
