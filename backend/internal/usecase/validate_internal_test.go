package usecase

import "testing"

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"História do Brasil":        "historia-do-brasil",
		"  Escândalos   políticos!": "escandalos-politicos",
		"Copa 2014 — São Paulo":     "copa-2014-sao-paulo",
		"!!!":                       "timeline",
	} {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
