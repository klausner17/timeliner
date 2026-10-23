package domain

import (
	"errors"
	"fmt"
)

// Kind classifica um erro de negócio. O controller traduz cada Kind para um status HTTP.
type Kind string

const (
	KindValidation   Kind = "invalid"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
)

// Error é o erro que os usecases devolvem quando a culpa é da requisição, não do sistema.
// Message vai para a tela do usuário: frase em português, sem detalhes internos.
type Error struct {
	Kind    Kind
	Message string
	// Fields traz o erro de cada campo em erros de validação.
	Fields map[string]string
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Kind, e.Message) }

func NotFound(what string) error {
	return &Error{Kind: KindNotFound, Message: what + " não encontrado(a)"}
}
func Forbidden(msg string) error    { return &Error{Kind: KindForbidden, Message: msg} }
func Conflict(msg string) error     { return &Error{Kind: KindConflict, Message: msg} }
func Unauthorized(msg string) error { return &Error{Kind: KindUnauthorized, Message: msg} }

// Invalid monta um erro de validação com os campos que falharam.
func Invalid(fields map[string]string) error {
	return &Error{Kind: KindValidation, Message: "Confira os campos destacados.", Fields: fields}
}

// ErrNotFound é o erro que os repositories devolvem quando não acham o registro.
// Os usecases trocam por NotFound com o nome do recurso.
var ErrNotFound = errors.New("registro não encontrado")

// ErrDuplicate é o erro dos repositories para chave única repetida.
var ErrDuplicate = errors.New("registro duplicado")

// KindOf devolve o Kind de err, ou "" se não for um erro de negócio.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return ""
}
