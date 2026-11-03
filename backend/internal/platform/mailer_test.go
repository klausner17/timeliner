package platform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

func TestResendSend(t *testing.T) {
	var got map[string]any
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"x"}`))
	}))
	defer srv.Close()
	m := Resend{APIKey: "re_teste", From: "Timeliner <ola@exemplo.com>", Endpoint: srv.URL}
	if err := m.Send(context.Background(), domain.Email{To: "ana@exemplo.com", Subject: "Oi", Text: "t", HTML: "<p>t</p>"}); err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer re_teste" || got["from"] != "Timeliner <ola@exemplo.com>" || got["subject"] != "Oi" {
		t.Fatalf("auth=%q corpo=%v", auth, got)
	}
	if to, _ := got["to"].([]any); len(to) != 1 || to[0] != "ana@exemplo.com" {
		t.Fatalf("to = %v", got["to"])
	}
}

func TestResendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"domínio não verificado"}`, http.StatusForbidden)
	}))
	defer srv.Close()
	err := Resend{APIKey: "k", From: "a@b.c", Endpoint: srv.URL}.Send(context.Background(), domain.Email{To: "x@y.z"})
	if err == nil {
		t.Fatal("esperava erro")
	}
}
