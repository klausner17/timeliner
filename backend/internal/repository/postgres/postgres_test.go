package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/repository/postgres"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

// Os fluxos completos rodam nos testes de usecase com TEST_DATABASE_URL; aqui ficam
// só os detalhes do Postgres.
func setup(t *testing.T) *usecase.Deps {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("defina TEST_DATABASE_URL para rodar contra o Postgres")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE users, timelines CASCADE`); err != nil {
		t.Fatal(err)
	}
	return postgres.New(pool)
}

func user(id string) *domain.User {
	return &domain.User{ID: id, Handle: id, Name: id, Email: id + "@x.com", PasswordHash: "h", Tint: domain.TintNeutral, CreatedAt: time.Now()}
}

func TestTxRollsBackOnError(t *testing.T) {
	d := setup(t)
	ctx := context.Background()
	boom := errors.New("boom")
	err := d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := d.Users.Create(ctx, user("ana")); err != nil {
			return err
		}
		if _, err := d.Users.ByID(ctx, "ana"); err != nil {
			t.Fatalf("dentro da transação devia achar: %v", err)
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if _, err := d.Users.ByID(ctx, "ana"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("rollback falhou: %v", err)
	}
}

func TestTxRollsBackOnPanic(t *testing.T) {
	d := setup(t)
	ctx := context.Background()
	func() {
		defer func() { _ = recover() }()
		_ = d.Tx.WithinTx(ctx, func(ctx context.Context) error {
			_ = d.Users.Create(ctx, user("ana"))
			panic("x")
		})
	}()
	if _, err := d.Users.ByID(ctx, "ana"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("rollback falhou: %v", err)
	}
}

func TestTxCommitsAndNests(t *testing.T) {
	d := setup(t)
	ctx := context.Background()
	err := d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		return d.Tx.WithinTx(ctx, func(ctx context.Context) error { return d.Users.Create(ctx, user("ana")) })
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Users.ByID(ctx, "ana"); err != nil {
		t.Fatalf("commit falhou: %v", err)
	}
}

func TestErrorsTranslate(t *testing.T) {
	d := setup(t)
	ctx := context.Background()
	if err := d.Users.Create(ctx, user("ana")); err != nil {
		t.Fatal(err)
	}
	if err := d.Users.Create(ctx, user("ana")); !errors.Is(err, domain.ErrDuplicate) {
		t.Fatalf("duplicado = %v", err)
	}
	if err := d.Events.Delete(ctx, "nada"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("delete inexistente = %v", err)
	}
}

func TestListQueryEscapesLike(t *testing.T) {
	d := setup(t)
	ctx := context.Background()
	if err := d.Users.Create(ctx, user("ana")); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for i, title := range []string{"100% Brasil", "1000 Brasil"} {
		tl := &domain.Timeline{ID: title, Slug: string(rune('a' + i)), Title: title, Category: "História", Color: "a", OwnerID: "ana",
			Visibility: domain.VisibilityPublic, Contributions: domain.ContributionsSuggestions, CreatedAt: now, UpdatedAt: now}
		if err := d.Timelines.Create(ctx, tl); err != nil {
			t.Fatal(err)
		}
	}
	got, total, err := d.Timelines.List(ctx, usecase.TimelineFilter{Query: "100%", Limit: 10})
	if err != nil || total != 1 || got[0].Title != "100% Brasil" {
		t.Fatalf("busca = %v %d %v", got, total, err)
	}
}
