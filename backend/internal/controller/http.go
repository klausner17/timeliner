// Package controller é a camada HTTP: lê a requisição, chama o usecase e converte
// o Output (ou o erro) para o JSON de api/openapi.yaml.
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/klausner17/timeliner/backend/internal/domain"
)

const maxBody = 1 << 20

type errorBody struct {
	Error errorJSON `json:"error"`
}

type errorJSON struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

var statusOf = map[domain.Kind]int{
	domain.KindValidation:   http.StatusBadRequest,
	domain.KindUnauthorized: http.StatusUnauthorized,
	domain.KindForbidden:    http.StatusForbidden,
	domain.KindNotFound:     http.StatusNotFound,
	domain.KindConflict:     http.StatusConflict,
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, msg string, fields map[string]string) {
	writeJSON(w, status, errorBody{errorJSON{Code: code, Message: msg, Fields: fields}})
}

// fail responde com o erro. Erros de negócio levam a mensagem do usecase; qualquer
// outro erro vira 500 com mensagem genérica, e o detalhe fica só no log.
func (c *Controller) fail(w http.ResponseWriter, r *http.Request, err error) {
	var de *domain.Error
	if errors.As(err, &de) {
		writeError(w, statusOf[de.Kind], string(de.Kind), de.Message, de.Fields)
		return
	}
	if errors.Is(err, context.Canceled) {
		return
	}
	c.log.ErrorContext(r.Context(), "erro interno", "method", r.Method, "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "Algo deu errado do nosso lado. Tente de novo em instantes.", nil)
}

// decode lê o corpo JSON. Erros de formato viram erros de validação legíveis.
func decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBody))
	err := dec.Decode(dst)
	if err == nil {
		return nil
	}
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &typeErr) && typeErr.Field != "":
		return domain.Invalid(map[string]string{jsonPath(typeErr.Field): "Valor em formato inválido."})
	case errors.Is(err, io.EOF):
		return &domain.Error{Kind: domain.KindValidation, Message: "Envie os dados no corpo da requisição."}
	default:
		return &domain.Error{Kind: domain.KindValidation, Message: "Não foi possível ler os dados enviados."}
	}
}

// jsonPath tira do caminho do erro os structs embutidos (ex.: "timelineFieldsJSON.title"),
// que não existem no JSON.
func jsonPath(field string) string {
	parts := strings.Split(field, ".")
	out := parts[:0]
	for _, p := range parts {
		if !strings.HasSuffix(p, "JSON") {
			out = append(out, p)
		}
	}
	return strings.Join(out, ".")
}

// queryInt lê um inteiro opcional da query string.
func queryInt(r *http.Request, name string, fields map[string]string) *int {
	s := r.URL.Query().Get(name)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		fields[name] = "Use um número inteiro."
		return nil
	}
	return &n
}

func queryBool(r *http.Request, name string, fields map[string]string) bool {
	switch r.URL.Query().Get(name) {
	case "", "false", "0":
		return false
	case "true", "1":
		return true
	}
	fields[name] = "Use true ou false."
	return false
}

func fieldsErr(fields map[string]string) error {
	if len(fields) == 0 {
		return nil
	}
	return domain.Invalid(fields)
}

type actorKey struct{}

// actor é o ID do usuário logado, ou "" sem login.
func actor(r *http.Request) string {
	id, _ := r.Context().Value(actorKey{}).(string)
	return id
}

// authenticate lê o token Bearer, se houver. Token inválido responde 401 já aqui,
// para o front saber que a sessão acabou; sem token a requisição segue anônima.
func (c *Controller) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" {
			next.ServeHTTP(w, r)
			return
		}
		token, ok := strings.CutPrefix(h, "Bearer ")
		if !ok || token == "" {
			c.fail(w, r, domain.Unauthorized("Envie o token como Authorization: Bearer <token>."))
			return
		}
		id, err := c.uc.Authenticate.Execute(r.Context(), token)
		if err != nil {
			c.fail(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, id)))
	})
}

// recoverer transforma um panic num 500 sem derrubar o servidor.
func (c *Controller) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				c.log.ErrorContext(r.Context(), "panic", "method", r.Method, "path", r.URL.Path, "panic", p)
				writeError(w, http.StatusInternalServerError, "internal", "Algo deu errado do nosso lado. Tente de novo em instantes.", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Optional distingue campo ausente (Set=false) de null (Set=true, Value=nil) nos PATCH.
type Optional[T any] struct {
	Set   bool
	Value *T
}

func (o *Optional[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// ptr devolve nil se o campo não veio, ou um ponteiro para o valor (que pode ser nil).
func (o Optional[T]) ptr() **T {
	if !o.Set {
		return nil
	}
	return &o.Value
}
