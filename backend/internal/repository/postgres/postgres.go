// Package postgres implementa os repositories com PostgreSQL (pgx) e roda as
// migrations com goose.
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/klausner17/timeliner/backend/internal/domain"
	"github.com/klausner17/timeliner/backend/internal/usecase"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate aplica as migrations que faltam.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	return migrate(ctx, db)
}

func migrate(ctx context.Context, db *sql.DB) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, db, mustSub(migrations, "migrations"))
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}

// New devolve os repositories e o TxManager sobre o pool.
func New(pool *pgxpool.Pool) *usecase.Deps {
	db := &db{pool}
	return &usecase.Deps{
		Users: users{db}, Sessions: sessions{db}, Timelines: timelines{db}, Events: events{db},
		Suggestions: suggestions{db}, Follows: follows{db}, Comparisons: comparisons{db}, Tx: txManager{pool},
	}
}

// querier é o que pool e transação têm em comum.
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type db struct{ pool *pgxpool.Pool }

type txKey struct{}

// q devolve a transação do ctx, se houver, ou o pool.
func (d *db) q(ctx context.Context) querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return d.pool
}

type txManager struct{ pool *pgxpool.Pool }

// WithinTx abre uma transação (ou reaproveita a que já está no ctx), faz commit se
// fn der certo e rollback se der erro ou panic.
func (m txManager) WithinTx(ctx context.Context, fn func(context.Context) error) (err error) {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	if err = fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// translate troca os erros do pgx pelos erros que os usecases conhecem.
func translate(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: %s", domain.ErrDuplicate, pgErr.ConstraintName)
	}
	return err
}

// mustAffect devolve ErrNotFound quando um UPDATE ou DELETE não achou a linha.
func mustAffect(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
