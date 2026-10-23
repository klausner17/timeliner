// Comando seed-grande cria uma timeline com muitos eventos, para testar paginação,
// faixas e desempenho. Usa o mesmo DATABASE_URL da API e cria o usuário @seed
// (senha "senha-forte") se ele ainda não existir.
//
//	go run ./cmd/seed-grande -n 20000 -slug grande
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/platform"
	"github.com/klausner17/timeliner/backend/internal/repository/postgres"
)

func main() {
	n := flag.Int("n", 20000, "quantos eventos criar")
	slug := flag.String("slug", "grande", "slug da timeline (não pode existir)")
	from := flag.Int("de", -3000, "primeiro ano")
	to := flag.Int("ate", 2025, "último ano")
	flag.Parse()
	if err := run(*n, *slug, *from, *to); err != nil {
		fmt.Fprintln(os.Stderr, "seed-grande:", err)
		os.Exit(1)
	}
}

func run(n int, slug string, from, to int) error {
	ctx := context.Background()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://timeliner:timeliner@localhost:5432/timeliner?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool); err != nil {
		return err
	}
	d := postgres.New(pool)
	clock, ids := platform.SystemClock{}, platform.RandomIDs{}

	owner, err := d.Users.ByHandle(ctx, "seed")
	if errors.Is(err, domain.ErrNotFound) {
		hash, _ := platform.Bcrypt{}.Hash("senha-forte")
		owner = &domain.User{ID: ids.NewID("us"), Handle: "seed", Name: "Seed", Email: "seed@exemplo.com", PasswordHash: hash, Tint: domain.TintNeutral, CreatedAt: clock.Now()}
		err = d.Users.Create(ctx, owner)
	}
	if err != nil {
		return err
	}

	now := clock.Now()
	t := domain.Timeline{
		ID: ids.NewID("tl"), Slug: slug, Title: fmt.Sprintf("Timeline grande (%d eventos)", n), Category: "História",
		Color: domain.ColorA, OwnerID: owner.ID, Visibility: domain.VisibilityPublic, Contributions: domain.ContributionsClosed,
		Eras: []domain.Era{}, Tags: []string{"teste"}, CreatedAt: now, UpdatedAt: now,
	}
	if err := d.Timelines.Create(ctx, &t); err != nil {
		return err
	}

	// Mais eventos perto do fim, como numa timeline histórica de verdade.
	r := rand.New(rand.NewPCG(1, 2))
	span := float64(to - from)
	const chunk = 5000
	for done := 0; done < n; done += chunk {
		batch := make([]domain.Event, min(chunk, n-done))
		for i := range batch {
			year := to - int(span*r.Float64()*r.Float64()*r.Float64())
			batch[i] = domain.Event{
				EventData: domain.EventData{Year: year, Title: fmt.Sprintf("Evento %d", done+i+1), Description: "Gerado pelo seed-grande."},
				ID:        ids.NewID("ev"), TimelineID: t.ID, AuthorID: owner.ID, CreatedAt: now,
			}
		}
		if err := d.Events.CreateMany(ctx, batch); err != nil {
			return err
		}
	}
	fmt.Printf("criada /t/%s com %d eventos de %d a %d\n", slug, n, from, to)
	return nil
}
