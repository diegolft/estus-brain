# Contexto de build: backend/ (é assim que o docker-compose chama:
# `build: ./backend`). Todos os COPY abaixo são relativos a backend/, não à
# raiz do repo.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/api ./cmd/api

FROM alpine:3.20
RUN adduser -D -u 10001 estus
WORKDIR /app
COPY --from=build /bin/api ./api
# O binário aplica as migrations sozinho no boot, lendo o caminho relativo
# "migrations" a partir do WORKDIR.
COPY --from=build /src/migrations ./migrations
# DOCUMENTS_DIR e ASSISTANT_DIR são relativos ao WORKDIR e o processo cria os
# diretórios no primeiro uso — precisam pertencer ao usuário não-root, senão
# o upload de documentos e o token do MCP falham com permission denied.
# Em produção monte um volume por cima de /app/data para não perder o acervo.
RUN mkdir -p /app/data/documents /app/data/assistant && chown -R estus:estus /app/data
USER estus
# Documental apenas. A porta real vem da env PORT (config.go usa 8080 como
# padrão) — é assim que o Cubeship injeta a porta do container.
EXPOSE 8080
ENTRYPOINT ["./api"]
