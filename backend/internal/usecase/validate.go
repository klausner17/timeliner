package usecase

import (
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"golang.org/x/text/unicode/norm"
)

// fieldErrors junta erros de validação por campo.
type fieldErrors map[string]string

func (f fieldErrors) add(field, msg string) {
	if _, ok := f[field]; !ok {
		f[field] = msg
	}
}

func (f fieldErrors) check(ok bool, field, msg string) {
	if !ok {
		f.add(field, msg)
	}
}

func (f fieldErrors) err() error {
	if len(f) == 0 {
		return nil
	}
	return domain.Invalid(f)
}

func length(s string, minLen, maxLen int) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(s))
	return n >= minLen && n <= maxLen
}

var handleRe = regexp.MustCompile(`^[a-z0-9_]{3,30}$`)

func validEmail(s string) bool {
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s
}

func validURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

const (
	minYear = -100000
	maxYear = 3000
)

// validateEvent confere os campos de um evento. prefix vai na frente dos nomes dos campos
// (ex.: "event." numa sugestão, "events[2]." na criação da timeline).
func validateEvent(f fieldErrors, prefix string, e domain.EventData, eras []domain.Era, r yearRules) {
	f.check(e.Year >= r.Lo && e.Year <= r.Hi, prefix+"year", "Ano fora do intervalo.")
	if r.YearOnly {
		f.check(e.Month == nil, prefix+"month", "Timelines com calendário próprio usam só o ano.")
	}
	f.check(length(e.Title, 1, 160), prefix+"title", "O título precisa ter de 1 a 160 caracteres.")
	f.check(length(e.Description, 0, 4000), prefix+"description", "A descrição pode ter até 4000 caracteres.")
	validateMoment(f, prefix, e)
	if e.EndYear != nil {
		f.check(*e.EndYear >= e.Year && *e.EndYear <= r.Hi, prefix+"endYear", "O fim precisa ser depois do início.")
	}
	if e.SourceURL != nil {
		f.check(validURL(*e.SourceURL), prefix+"sourceUrl", "Link inválido.")
	}
	if e.Era != nil {
		found := false
		for _, era := range eras {
			found = found || era.Name == *e.Era
		}
		f.check(found, prefix+"era", "Período não existe nesta timeline.")
	}
}

var timeRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// validateMoment confere mês, dia e hora: cada um só vale com o anterior preenchido.
func validateMoment(f fieldErrors, prefix string, e domain.EventData) {
	if e.Month != nil {
		f.check(*e.Month >= 1 && *e.Month <= 12, prefix+"month", "Mês inválido.")
	}
	if e.Day != nil {
		f.check(e.Month != nil, prefix+"day", "Informe o mês antes do dia.")
		f.check(e.Month == nil || validDay(e.Year, *e.Month, *e.Day), prefix+"day", "Este dia não existe no mês.")
	}
	if e.Time != nil {
		f.check(e.Day != nil, prefix+"time", "Informe o dia antes da hora.")
		f.check(timeRe.MatchString(*e.Time), prefix+"time", "Use a hora no formato 14:30.")
	}
}

// validDay diz se o dia existe no mês daquele ano, no calendário gregoriano.
func validDay(year, month, day int) bool {
	if day < 1 || day > 31 {
		return false
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC).Day() == day
}

// normalizeEvent limpa espaços e troca textos vazios opcionais por nil.
func normalizeEvent(e domain.EventData) domain.EventData {
	e.Title = strings.TrimSpace(e.Title)
	e.Description = strings.TrimSpace(e.Description)
	e.Time = emptyToNil(e.Time)
	e.SourceURL = emptyToNil(e.SourceURL)
	e.Era = emptyToNil(e.Era)
	return e
}

func emptyToNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// slugify transforma "História do Brasil" em "historia-do-brasil".
func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		switch {
		case unicode.Is(unicode.Mn, r):
			// acento separado pela normalização
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.TrimSuffix(b.String(), "-")
	if len(out) > 60 {
		out = strings.TrimSuffix(out[:60], "-")
	}
	if out == "" {
		out = "timeline"
	}
	return out
}
