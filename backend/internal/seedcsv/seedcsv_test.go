package seedcsv_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/platform"
	"github.com/klausner17/timeliner/backend/internal/repository/memory"
	"github.com/klausner17/timeliner/backend/internal/seedcsv"
)

// Os CSVs de dados/brasil precisam passar pelas mesmas validações da API.
func TestDadosBrasil(t *testing.T) {
	seeds, err := seedcsv.Load("../../../dados/brasil")
	if err != nil {
		t.Fatal(err)
	}
	if len(seeds) == 0 {
		t.Fatal("nenhuma timeline lida")
	}
	for _, s := range seeds {
		for i, e := range s.Events {
			if e.SourceURL == nil || !strings.HasPrefix(*e.SourceURL, "https://pt.wikipedia.org/wiki/") {
				t.Errorf("%s evento %d (%s): fonte precisa ser um link da Wikipédia", s.Slug, i+1, e.Title)
			}
		}
	}

	ctx := context.Background()
	d := memory.New()
	d.Clock, d.IDs = platform.SystemClock{}, platform.RandomIDs{}
	u := &domain.User{ID: "us_1", Handle: "dono", Name: "Dono", Email: "dono@exemplo.com", Tint: domain.TintNeutral}
	if err := d.Users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	res, err := seedcsv.Import(ctx, d, u.ID, seeds, false)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range res {
		if r.Skipped || r.Slug != seeds[i].Slug {
			t.Errorf("esperava criar /t/%s, veio %+v", seeds[i].Slug, r)
		}
	}

	// De novo: pula as que já existem; com substituir, recria.
	res, err = seedcsv.Import(ctx, d, u.ID, seeds[:1], false)
	if err != nil || !res[0].Skipped {
		t.Errorf("esperava pular a existente: %+v %v", res, err)
	}
	res, err = seedcsv.Import(ctx, d, u.ID, seeds[:1], true)
	if err != nil || res[0].Skipped || res[0].Slug != seeds[0].Slug {
		t.Errorf("esperava recriar com o mesmo slug: %+v %v", res, err)
	}
}

func TestSlugPrecisaBaterComTitulo(t *testing.T) {
	dir := t.TempDir()
	write(t, dir+"/timelines.csv", "slug,titulo,descricao,categoria,cor,tags,visibilidade,contribuicoes\noutro,Meu título,,História,a,,,\n")
	if _, err := seedcsv.Load(dir); err == nil || !strings.Contains(err.Error(), "meu-titulo") {
		t.Errorf("esperava erro sugerindo o slug meu-titulo, veio %v", err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
