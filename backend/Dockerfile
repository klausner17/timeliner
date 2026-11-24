# Build da API em Go e imagem final enxuta com o binário.
FROM golang:1.24-alpine AS build
WORKDIR /src
# Cache mounts do BuildKit: módulos e build cache do Go sobrevivem entre builds, fora
# das camadas. Sem eles, qualquer mudança no código recompila todas as dependências
# (~40s contra poucos segundos incremental).
COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.22
RUN adduser -D -H -u 10001 app
COPY --from=build /out/api /usr/local/bin/api
USER app
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
