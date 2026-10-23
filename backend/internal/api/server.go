// Package api monta as rotas HTTP do Timeliner.
package api

import (
	"encoding/json"
	"net/http"
)

// NewHandler devolve o roteador com todas as rotas da API, sob /api.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", health)
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
