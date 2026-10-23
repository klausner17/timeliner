# Backend do Timeliner

API em Go com PostgreSQL. O contrato é [`api/openapi.yaml`](../api/openapi.yaml), na raiz do repo.

## Camadas

```
cmd/api/              main: lê a configuração, abre o banco, roda as migrations e sobe o HTTP
cmd/seed-grande/      cria uma timeline com milhares de eventos, para testar timelines grandes
internal/domain/      entidades (Timeline, Event, Suggestion...) e erros de negócio
internal/usecase/     regras de negócio; um tipo por caso de uso, todos UseCase[I, O]
internal/repository/  postgres/ (pgx + goose) e memory/ (para testes)
internal/controller/  HTTP (chi): lê a requisição, chama o usecase, converte para JSON
internal/platform/    bcrypt, relógio e geração de IDs
```

- **UseCase.** Todo caso de uso segue `UseCase[I, O any] interface { Execute(ctx, I) (O, error) }`
  (`internal/usecase/usecase.go`) e tem só esse método. Os Inputs e Outputs são definidos
  no próprio usecase. Quem está logado chega no Input como `ActorID`, e o usecase decide a permissão.
- **Repositories.** As interfaces ficam em `internal/usecase/ports.go`, ao lado de quem as usa.
  Devolvem `domain.ErrNotFound` e `domain.ErrDuplicate`, e o usecase traduz para erros de negócio.
- **Transações.** `TxManager.WithinTx(ctx, fn)` roda `fn` numa transação. A implementação do
  Postgres põe o `pgx.Tx` no `ctx`, e os repositories usam a transação quando ela existe. Aprovar e
  desfazer sugestões, criar timeline com eventos e cadastrar usuário usam transação.
- **Controller.** Converte o Output para o JSON do contrato (`dto.go`) e o erro para o status HTTP.

## Timelines grandes

Nenhuma rota devolve todos os eventos de uma vez:

- `GET /timelines/{slug}/events` pagina por cursor (keyset sobre `year, fim, created_at, id`, coberto
  pelo índice `events_timeline_order_idx`), 100 por página por padrão e até 500.
- `GET /timelines/{slug}` traz só a primeira página e o `eventsNextCursor`.
- `GET /timelines/{slug}/summary` e cada trilha de `GET /compare` devolvem os eventos do trecho se
  forem até `maxEvents` (150), e senão faixas de anos com contagem e 3 exemplos (`mode: faixas`).
- Os `points` dos cartões saem de posições já reduzidas no banco (no máximo 1001 valores).
- A criação de timeline aceita até 500 eventos, gravados com `COPY`.

```bash
DATABASE_URL=... go run ./cmd/seed-grande -n 20000 -slug grande   # cria /t/grande
```

## Erros

Toda resposta de erro tem o formato `{"error": {"code", "message", "fields?"}}`:

- `message` é uma frase em português pronta para mostrar ao usuário, escrita no usecase (`domain.Error`).
- `fields` traz a mensagem de cada campo com erro, pelo caminho no JSON (`title`, `events[2].year`).
- Erros que não são de negócio (banco, bug, panic) viram `500` com `code: internal` e mensagem
  genérica. O detalhe vai só para o log.

| code | status |
| --- | --- |
| `invalid` | 400 |
| `unauthorized` | 401 |
| `forbidden` | 403 |
| `not_found` | 404 |
| `conflict` | 409 |
| `internal` | 500 |

## Configuração

| Variável | Padrão |
| --- | --- |
| `PORT` | `8080` |
| `DATABASE_URL` | `postgres://timeliner:timeliner@localhost:5432/timeliner?sslmode=disable` |
| `MIGRATE` | as migrations rodam na subida; `false` pula essa etapa |

As migrations ficam em `internal/repository/postgres/migrations/`, embutidas no binário.

## Testes

```bash
go test ./...   # usecases e controllers com o repositório em memória
TEST_DATABASE_URL=postgres://timeliner:timeliner@localhost:5432/timeliner_test?sslmode=disable \
  go test -count=1 -p 1 ./...   # os mesmos testes de usecase, mais os do repositório, contra o Postgres
```

O banco de `TEST_DATABASE_URL` é apagado a cada teste. Use um banco só para testes.
