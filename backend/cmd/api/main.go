// Comando api sobe o servidor HTTP do Timeliner.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/klausner17/timeliner/backend/internal/api"
)

func main() {
	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           api.NewHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("Timeliner API em http://localhost%s", addr)
	log.Fatal(srv.ListenAndServe())
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
