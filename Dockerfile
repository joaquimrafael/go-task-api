# syntax=docker/dockerfile:1

FROM golang:1.26.4-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/task-api \
    ./cmd/api

FROM alpine:3.23

RUN addgroup -S api \
    && adduser -S -G api api \
    && mkdir -p /data \
    && chown api:api /data

COPY --from=build /out/task-api /usr/local/bin/task-api

ENV LISTEN_ADDR=:8080 \
    DATABASE_PATH=/data/tasks.db

EXPOSE 8080

USER api

ENTRYPOINT ["/usr/local/bin/task-api"]
