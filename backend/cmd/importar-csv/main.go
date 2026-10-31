// Comando importar-csv cria no banco as timelines de uma pasta de CSVs (veja o pacote
// seedcsv para o formato). Usa o mesmo DATABASE_URL da API, aplica as migrations e cria
// as timelines em nome de um usuário que já existe.
//
//	go run ./cmd/importar-csv                                  # tudo de ../dados/brasil, dono @klausner
//	go run ./cmd/importar-csv -dono seed -so biografia-de-lula  # só uma
//	go run ./cmd/importar-csv -substituir                      # apaga e recria as que já existem
//	go run ./cmd/importar-csv -testar                          # só valida, sem banco
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/platform"
	"github.com/klausner17/timeliner/backend/internal/repository/memory"
	"github.com/klausner17/timeliner/backend/internal/repository/postgres"
	"github.com/klausner17/timeliner/backend/internal/seedcsv"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

func main() {
	dir := flag.String("dir", "../dados/brasil", "pasta com timelines.csv, periodos/ e eventos/")
	owner := flag.String("dono", "klausner", "handle do usuário dono das timelines (precisa existir)")
	only := flag.String("so", "", "slugs separados por vírgula; vazio importa todas")
	replace := flag.Bool("substituir", false, "apaga e recria as timelines que já existem")
	dry := flag.Bool("testar", false, "valida os CSVs num banco em memória, sem tocar no Postgres")
	flag.Parse()
	if err := run(*dir, *owner, *only, *replace, *dry); err != nil {
		fmt.Fprintln(os.Stderr, "importar-csv:", err)
		os.Exit(1)
	}
}

func run(dir, owner, only string, replace, dry bool) error {
	var slugs []string
	for _, s := range strings.Split(only, ",") {
		if s = strings.TrimSpace(s); s != "" {
			slugs = append(slugs, s)
		}
	}
	seeds, err := seedcsv.Load(dir, slugs...)
	if err != nil {
		return err
	}

	ctx := context.Background()
	var d *usecase.Deps
	if dry {
		d = memory.New()
		u := &domain.User{ID: "us_teste", Handle: owner, Name: owner, Email: owner + "@exemplo.com", Tint: domain.TintNeutral}
		if err := d.Users.Create(ctx, u); err != nil {
			return err
		}
	} else {
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
		d = postgres.New(pool)
	}
	d.Hasher, d.Clock, d.IDs = platform.Bcrypt{}, platform.SystemClock{}, platform.RandomIDs{}

	u, err := d.Users.ByHandle(ctx, owner)
	if errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("usuário @%s não existe; crie a conta antes ou use -dono", owner)
	}
	if err != nil {
		return err
	}
	results, err := seedcsv.Import(ctx, d, u.ID, seeds, replace)
	for _, r := range results {
		if r.Skipped {
			fmt.Printf("  já existe  /t/%s (use -substituir para recriar)\n", r.Slug)
		} else {
			fmt.Printf("  criada     /t/%s com %d eventos\n", r.Slug, r.Events)
		}
	}
	if err != nil {
		return err
	}
	if dry {
		fmt.Println("CSVs válidos; nada foi gravado.")
	}
	return nil
}
