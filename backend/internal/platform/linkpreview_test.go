package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLinkPreview(t *testing.T) {
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/og", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head>
			<title>Título reserva</title>
			<meta property="og:title" content="Lei Áurea &amp; abolição">
			<meta property="og:description" content="  Assinada em
			13 de maio de 1888. ">
			<meta property="og:image" content="/img/lei.jpg">
			<meta property="og:site_name" content="Arquivo">
			</head><body><meta property="og:title" content="não lê o body"></body></html>`))
	})
	mux.HandleFunc("/simples", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=iso-8859-1")
		// "Abolição" em Latin-1.
		_, _ = w.Write([]byte("<html><head><title>Aboli\xe7\xe3o</title><meta name=\"description\" content=\"Resumo\"></head></html>"))
	})
	mux.HandleFunc("/foto.png", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("\x89PNG"))
	})
	mux.HandleFunc("/pdf", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
	})
	mux.Handle("/volta", http.RedirectHandler("/og", http.StatusFound))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	l := NewLinkPreviews(true)
	ctx := context.Background()

	p, err := l.Preview(ctx, srv.URL+"/volta")
	if err != nil {
		t.Fatal(err)
	}
	if p.URL != srv.URL+"/volta" || p.Title != "Lei Áurea & abolição" || p.Description != "Assinada em 13 de maio de 1888." ||
		p.Image != srv.URL+"/img/lei.jpg" || p.SiteName != "Arquivo" {
		t.Fatalf("prévia = %+v", p)
	}
	if _, err := l.Preview(ctx, srv.URL+"/volta"); err != nil || hits != 1 {
		t.Fatalf("devia vir do cache: hits=%d err=%v", hits, err)
	}

	p, err = l.Preview(ctx, srv.URL+"/simples")
	if err != nil || p.Title != "Abolição" || p.Description != "Resumo" || p.Image != "" || !strings.HasPrefix(p.SiteName, "127.0.0.1") {
		t.Fatalf("prévia = %+v, err = %v", p, err)
	}

	p, err = l.Preview(ctx, srv.URL+"/foto.png")
	if err != nil || p.Image != srv.URL+"/foto.png" {
		t.Fatalf("imagem = %+v, err = %v", p, err)
	}

	if _, err := l.Preview(ctx, srv.URL+"/pdf"); err == nil {
		t.Fatal("PDF não tem prévia")
	}
	if _, err := l.Preview(ctx, "ftp://exemplo.com/x"); err == nil {
		t.Fatal("só http e https")
	}
}

func TestLinkPreviewBlocksInternalNetwork(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<title>segredo</title>"))
	}))
	defer srv.Close()
	l := NewLinkPreviews(false)
	if p, err := l.Preview(context.Background(), srv.URL); err == nil {
		t.Fatalf("buscou a rede interna: %+v", p)
	}
	if p, err := l.Preview(context.Background(), strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)); err == nil {
		t.Fatalf("buscou localhost: %+v", p)
	}
}
