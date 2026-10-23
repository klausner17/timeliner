package usecase_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klausner17/timeliner/backend/internal/repository/memory"
	"github.com/klausner17/timeliner/backend/internal/repository/postgres"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

// Com TEST_DATABASE_URL, os mesmos testes rodam contra o Postgres em vez da memória.
// O banco é limpo a cada teste.
var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		var err error
		if pool, err = pgxpool.New(context.Background(), url); err == nil {
			err = postgres.Migrate(context.Background(), pool)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "banco de teste:", err)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}

func repositories() *usecase.Deps {
	if pool == nil {
		return memory.New()
	}
	_, err := pool.Exec(context.Background(), `TRUNCATE users, sessions, timelines, events, suggestions,
		timeline_follows, user_follows, saved_comparisons CASCADE`)
	if err != nil {
		panic(err)
	}
	return postgres.New(pool)
}
