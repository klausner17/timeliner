package main

import "testing"

func TestAppURLFrom(t *testing.T) {
	ok := []struct {
		raw     string
		sending bool
		want    string
	}{
		{"", false, "http://localhost:8000"},
		{"https://klausner.site/app/timeliner/", true, "https://klausner.site/app/timeliner"},
		{" http://localhost:5173 ", false, "http://localhost:5173"},
	}
	for _, c := range ok {
		got, err := appURLFrom(c.raw, c.sending)
		if err != nil || got != c.want {
			t.Errorf("appURLFrom(%q, %v) = %q, %v; queria %q", c.raw, c.sending, got, err, c.want)
		}
	}
	for _, raw := range []string{"", "klausner.site", "ftp://klausner.site", "https://", "https://x.com/?a=1"} {
		if _, err := appURLFrom(raw, true); err == nil {
			t.Errorf("appURLFrom(%q, true) devia falhar", raw)
		}
	}
}
