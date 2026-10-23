// Package usecase tem as regras de negócio do Timeliner. Cada caso de uso é um tipo
// com um único método, Execute, e depende só de domain e das interfaces de ports.go.
package usecase

import "context"

// UseCase é o contrato de todo caso de uso: recebe um Input e devolve um Output.
type UseCase[I, O any] interface {
	Execute(ctx context.Context, in I) (O, error)
}

// Empty é o Input ou Output de casos de uso que não precisam de dados.
type Empty struct{}
