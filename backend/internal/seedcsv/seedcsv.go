// Package seedcsv lê timelines prontas de arquivos CSV e as cria no banco pelo mesmo caso
// de uso da API, com as mesmas validações.
//
// Uma pasta de dados tem:
//
//	timelines.csv            slug,titulo,descricao,categoria,cor,tags,visibilidade,contribuicoes
//	periodos/<slug>.csv      nome,inicio,fim,cor                       (opcional)
//	eventos/<slug>.csv       data,ano_fim,hora,titulo,descricao,periodo,fonte
//
// Datas são AAAA, AAAA-MM ou AAAA-MM-DD (ano negativo para antes de Cristo). Tags vão
// separadas por "|". Um período sem fim continua até hoje.
package seedcsv

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

// Seed é uma timeline lida dos CSVs, pronta para virar CreateTimelineInput.
type Seed struct {
	Slug   string
	Fields usecase.TimelineFields
	Events []domain.EventData
}

var (
	timelineCols = []string{"slug", "titulo", "descricao", "categoria", "cor", "tags", "visibilidade", "contribuicoes"}
	periodCols   = []string{"nome", "inicio", "fim", "cor"}
	eventCols    = []string{"data", "ano_fim", "hora", "titulo", "descricao", "periodo", "fonte"}
)

// Load lê todas as timelines da pasta. only, se não for vazio, filtra pelos slugs.
func Load(dir string, only ...string) ([]Seed, error) {
	rows, err := readCSV(filepath.Join(dir, "timelines.csv"), timelineCols)
	if err != nil {
		return nil, err
	}
	var out []Seed
	seen := map[string]bool{}
	for i, r := range rows {
		where := fmt.Sprintf("timelines.csv linha %d", i+2)
		s, err := timelineOf(r)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", where, err)
		}
		if seen[s.Slug] {
			return nil, fmt.Errorf("%s: slug %q repetido", where, s.Slug)
		}
		seen[s.Slug] = true
		if len(only) > 0 && !contains(only, s.Slug) {
			continue
		}
		eras, err := loadPeriods(filepath.Join(dir, "periodos", s.Slug+".csv"))
		if err != nil {
			return nil, err
		}
		s.Fields.Eras = &eras
		if s.Events, err = loadEvents(filepath.Join(dir, "eventos", s.Slug+".csv")); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	for _, slug := range only {
		if !seen[slug] {
			return nil, fmt.Errorf("timeline %q não está em timelines.csv", slug)
		}
	}
	return out, nil
}

func timelineOf(r map[string]string) (Seed, error) {
	s := Seed{Slug: r["slug"]}
	title := r["titulo"]
	if want := usecase.Slugify(title); s.Slug != want {
		return s, fmt.Errorf("o slug de %q precisa ser %q (a API gera o slug pelo título)", title, want)
	}
	desc, cat := r["descricao"], r["categoria"]
	color := domain.TrackColor(orDefault(r["cor"], "a"))
	vis := domain.Visibility(orDefault(r["visibilidade"], "publica"))
	contrib := domain.ContributionMode(orDefault(r["contribuicoes"], "sugestoes"))
	tags := []string{}
	for _, t := range strings.Split(r["tags"], "|") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	s.Fields = usecase.TimelineFields{
		Title: &title, Description: &desc, Category: &cat, Color: &color,
		Visibility: &vis, Contributions: &contrib, Tags: &tags,
	}
	return s, nil
}

func loadPeriods(path string) ([]domain.Era, error) {
	rows, err := readCSV(path, periodCols)
	if errors.Is(err, os.ErrNotExist) {
		return []domain.Era{}, nil
	}
	if err != nil {
		return nil, err
	}
	eras := make([]domain.Era, len(rows))
	for i, r := range rows {
		where := fmt.Sprintf("%s linha %d", path, i+2)
		e := domain.Era{Name: r["nome"]}
		if e.Start, e.StartMonth, e.StartDay, err = parseDate(r["inicio"]); err != nil {
			return nil, fmt.Errorf("%s: inicio: %w", where, err)
		}
		if r["fim"] != "" {
			var end int
			if end, e.EndMonth, e.EndDay, err = parseDate(r["fim"]); err != nil {
				return nil, fmt.Errorf("%s: fim: %w", where, err)
			}
			e.End = &end
		}
		if c := r["cor"]; c != "" {
			pc := domain.PeriodColor(c)
			e.Color = &pc
		}
		eras[i] = e
	}
	return eras, nil
}

func loadEvents(path string) ([]domain.EventData, error) {
	rows, err := readCSV(path, eventCols)
	if err != nil {
		return nil, err
	}
	events := make([]domain.EventData, len(rows))
	for i, r := range rows {
		where := fmt.Sprintf("%s linha %d", path, i+2)
		var e domain.EventData
		if e.Year, e.Month, e.Day, err = parseDate(r["data"]); err != nil {
			return nil, fmt.Errorf("%s: data: %w", where, err)
		}
		if v := r["ano_fim"]; v != "" {
			end, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("%s: ano_fim %q não é um ano", where, v)
			}
			e.EndYear = &end
		}
		e.Time = optional(r["hora"])
		e.Title, e.Description = r["titulo"], r["descricao"]
		e.Era = optional(r["periodo"])
		e.SourceURL = optional(r["fonte"])
		events[i] = e
	}
	return events, nil
}

var dateRe = regexp.MustCompile(`^(-?\d{1,6})(?:-(\d{1,2})(?:-(\d{1,2}))?)?$`)

// parseDate lê AAAA, AAAA-MM ou AAAA-MM-DD.
func parseDate(s string) (year int, month, day *int, err error) {
	m := dateRe.FindStringSubmatch(s)
	if m == nil {
		return 0, nil, nil, fmt.Errorf("%q não é AAAA, AAAA-MM nem AAAA-MM-DD", s)
	}
	year, _ = strconv.Atoi(m[1])
	if m[2] != "" {
		v, _ := strconv.Atoi(m[2])
		month = &v
	}
	if m[3] != "" {
		v, _ := strconv.Atoi(m[3])
		day = &v
	}
	return year, month, day, nil
}

// readCSV lê o arquivo como uma lista de linhas indexadas pelo cabeçalho, que precisa ter
// exatamente as colunas esperadas (em qualquer ordem).
func readCSV(path string, cols []string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	header[0] = strings.TrimPrefix(header[0], "\ufeff")
	for _, c := range cols {
		if !contains(header, c) {
			return nil, fmt.Errorf("%s: falta a coluna %q (esperado: %s)", path, c, strings.Join(cols, ","))
		}
	}
	var out []map[string]string
	for line := 2; ; line++ {
		rec, err := r.Read()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		row := make(map[string]string, len(header))
		for i, h := range header {
			row[h] = strings.TrimSpace(rec[i])
		}
		out = append(out, row)
	}
}

// Result diz o que aconteceu com cada timeline na importação.
type Result struct {
	Slug   string
	Events int
	// Skipped é true quando a timeline já existia e não foi substituída.
	Skipped bool
}

// Import cria as timelines em nome de ownerID. Uma timeline cujo slug já existe é pulada,
// ou apagada e criada de novo com replace (só se for do mesmo dono).
func Import(ctx context.Context, d *usecase.Deps, ownerID string, seeds []Seed, replace bool) ([]Result, error) {
	create, del := usecase.NewCreateTimeline(d), usecase.NewDeleteTimeline(d)
	var out []Result
	for _, s := range seeds {
		existing, err := d.Timelines.BySlug(ctx, s.Slug)
		switch {
		case err == nil && !replace:
			out = append(out, Result{Slug: s.Slug, Skipped: true})
			continue
		case err == nil:
			if existing.OwnerID != ownerID {
				return out, fmt.Errorf("%s: já existe e é de outro usuário", s.Slug)
			}
			if _, err := del.Execute(ctx, usecase.SlugInput{ActorID: ownerID, Slug: s.Slug}); err != nil {
				return out, fmt.Errorf("%s: apagando a antiga: %w", s.Slug, err)
			}
		case !errors.Is(err, domain.ErrNotFound):
			return out, err
		}
		v, err := create.Execute(ctx, usecase.CreateTimelineInput{ActorID: ownerID, TimelineFields: s.Fields, Events: s.Events})
		if err != nil {
			var de *domain.Error
			if errors.As(err, &de) && len(de.Fields) > 0 {
				return out, fmt.Errorf("%s: %s %v", s.Slug, de.Message, de.Fields)
			}
			return out, fmt.Errorf("%s: %w", s.Slug, err)
		}
		out = append(out, Result{Slug: v.Timeline.Slug, Events: len(s.Events)})
	}
	return out, nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
