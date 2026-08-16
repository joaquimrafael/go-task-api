# Repository Guidelines

## Project Structure & Module Organization

This repository contains a small Go REST API. The executable entry point is `cmd/api/main.go`; keep startup, routing, and dependency wiring there. HTTP handlers live under `internal/handler/`, business rules under `internal/service/`, task types and domain errors under `internal/model/`, and SQLite access under `internal/repository/`.

Place tests beside the code they exercise, using Go's `_test.go` suffix. Runtime configuration comes from `LISTEN_ADDR` and `DATABASE_PATH`, with local defaults. Running the API creates a local `tasks.db` SQLite file. `Dockerfile` and `compose.yaml` provide a containerized workflow with persistent SQLite storage. Avoid committing generated binaries, database files, local environment files, or editor metadata.

## Build, Test, and Development Commands

- `go run ./cmd/api` starts the API locally on port `8080`.
- `curl http://localhost:8080/health` checks the server and database connection.
- `curl http://localhost:8080/tasks` lists the currently stored tasks.
- `go build ./...` compiles every package and catches build errors.
- `go test ./...` runs all repository tests.
- `go test -cover ./...` reports package-level test coverage.
- `go fmt ./...` applies standard Go formatting before review.
- `go vet ./...` checks for suspicious constructs not caught by compilation.
- `docker compose up --build -d` builds and starts the containerized API.
- `docker compose down` stops the containerized API without deleting its database volume.

## Coding Style & Naming Conventions

Follow idiomatic Go and let `gofmt` determine tabs, spacing, and import grouping. Package names should be short, lowercase, and singular. Exported identifiers use `PascalCase`; unexported identifiers use `camelCase`. Name handlers by purpose, such as `CreateTask` or `ListTasks`, and keep each function focused. Handle errors explicitly and add context when returning or logging them.

## Testing Guidelines

Use Go's standard `testing` package and `net/http/httptest` for handlers. Name tests `TestFunctionName` and prefer table-driven cases when validating multiple inputs or status codes. Cover success paths, malformed requests, unsupported methods, and storage failures. No coverage threshold is currently enforced; prioritize meaningful behavior over a percentage target.

## Commit & Pull Request Guidelines

Git history is not available in this working directory, so no existing convention can be inferred. Use short, imperative commits such as `add task creation handler`. Keep commits focused. Pull requests should explain the behavior changed, list verification commands, and link any related issue. Include example requests and responses when an API contract changes; screenshots are generally unnecessary.

## Learning-Focused Contributions

This is a learning project. When assisting, explain the relevant concept, break work into small steps, and let the learner implement meaningful changes. Prefer targeted hints and code review over replacing whole files unless explicitly requested.

## Current Implementation Progress

- Phases 0-10 are complete. The task model exists, and `OpenSQLite` registers the pure-Go SQLite driver, limits the pool to one connection, verifies startup access with a timeout, and initializes the `tasks` schema.
- `SQLiteTaskRepository` implements create, list, retrieve, update, and delete with context-aware, parameterized SQL and not-found translation.
- `TaskService` depends on its consumer-owned repository interface, validates and trims titles for create/update, delegates every operation, and preserves errors with `%w`. Domain sentinel errors intentionally live in `internal/model`.
- `TaskHandler` depends on its own consumer-owned service interface and implements list, retrieve, create, update, and delete. It strictly decodes exactly one non-null JSON object, rejects unknown fields, parses positive path IDs through a shared helper, and maps validation, not-found, and unexpected errors to 422, 404, and 500 responses. Shared helpers produce consistent JSON and JSON-error responses. Successful deletion returns 204 without a body or JSON content type.
- The health handler pings the database with a request-derived two-second timeout and returns 503 when the database is unavailable.
- `cmd/api/main.go` wires the database, repository, service, handlers, and a JSON `slog` request logger. Its extracted `newRouter` function owns the Go 1.22 method-aware routes for `GET /health`, `GET /tasks`, `GET /tasks/{id}`, `POST /tasks`, `PUT /tasks/{id}`, and `DELETE /tasks/{id}`. The API reads `LISTEN_ADDR` and `DATABASE_PATH` from the environment with `:8080` and `tasks.db` defaults, and its `http.Server` has read-header, read, write, and idle timeouts.
- Phase 9 request logging and graceful shutdown are complete. `requestLogger` wraps the full router and logs method, path, captured response status, and duration. Its `statusWriter` handles explicit status codes, implicit and empty 200 responses, duplicate `WriteHeader` calls, and exposes the wrapped writer through `Unwrap`.
- `main` uses `signal.NotifyContext` for SIGINT/SIGTERM and delegates the server lifecycle to the testable `serveUntilShutdown` helper. `ListenAndServe` runs concurrently, shutdown uses a fresh five-second timeout context, and `http.ErrServerClosed` is treated as an expected result. Manual CRUD, error-response, logging, and Ctrl+C shutdown checks pass.
- Repository integration tests use temporary SQLite files and cover initialization, CRUD success and failure paths, ordering, constraints, and canceled contexts. Table-driven service tests use a fake repository, and public-package handler tests use `httptest.NewRecorder` with fake services to cover every endpoint's happy path and representative 400/404/422/500/503 failures. Router tests cover all registered routes plus 404 and 405 behavior.
- A lightweight API integration test exercises the real router, handlers, service, repository, health check, and a temporary SQLite database through one complete CRUD lifecycle. It runs in-process without opening a network port.
- Middleware tests capture and decode JSON log output for empty and body-writing 200 responses, explicit 201, and explicit 204 while verifying method, path, status, and duration fields. Lifecycle tests use a fake server to cover normal shutdown, fresh deadline contexts, early server failures, shutdown failures, and expected `http.ErrServerClosed` results.
- `go test -race -cover ./...` passes with 87.6% overall statement coverage: `cmd/api` 63.3%, `internal/handler` 98.8%, `internal/repository` 88.4%, and `internal/service` 100.0%. `serveUntilShutdown` and every middleware function have 100% coverage. `go vet ./...` also passes.
- Phase 10 polish is complete: environment configuration has local defaults, generated SQLite files are ignored, the README documents local and containerized operation, and formatting, vet, race tests, and schema recreation checks pass.
- Packages and exported identifiers have concise Go doc comments focused on their purpose and non-obvious behavior.
- `Dockerfile` builds a static Linux binary in a multi-stage image and runs it as a non-root user. `compose.yaml` publishes the API, checks `/health`, and persists `/data/tasks.db` in the `task-data` volume. The static Linux build passes; a live Docker run remains environment-dependent because Docker is unavailable in the current workspace.

## Target API Specification

Treat the documentation in `specs/` as the detailed implementation guide and source of truth for the finished API. Build the core version in its numbered phases before attempting optional extensions.

- Use Go 1.22+ and `net/http` method-aware routes, with no web framework.
- Organize the application into `internal/model`, `internal/repository`, `internal/service`, and `internal/handler`. Handlers own HTTP concerns, services own validation and business rules, repositories own SQL and database-error translation, and `main` owns configuration, wiring, and lifecycle.
- Pass `context.Context` through HTTP, service, and repository boundaries. Define the repository interface in the consuming `service` package and inject dependencies through constructors.
- Persist tasks in SQLite through `database/sql` and the pure-Go `modernc.org/sqlite` driver. Open one shared database in `main`, ping at startup, set `SetMaxOpenConns(1)`, initialize the schema automatically, and close it at shutdown.
- Parameterize all SQL with `?`, select explicit columns (never `SELECT *`), order lists by ID, close and check query rows, and translate `sql.ErrNoRows` or zero affected rows into a domain not-found error.
- A task has `id`, `title`, `description`, `completed`, `created_at`, and `updated_at`. Keep client-writable input separate from stored output. Trim titles and require 1-120 characters; descriptions are optional.
- Implement `GET /tasks`, `GET /tasks/{id}`, `POST /tasks`, `PUT /tasks/{id}`, `DELETE /tasks/{id}`, and database-aware `GET /health`. Success codes are respectively 200, 200, 201, 200, 204, and 200; health returns 503 when the database is unavailable.
- JSON errors always use `{"error":"..."}`. JSON responses use `Content-Type: application/json`. Reject invalid positive IDs and malformed or extra request JSON with 400, validation failures with 422, missing tasks with 404, and unexpected failures with 500.
- Use a fake repository for service tests, `httptest` for handlers, and a real SQLite file under `t.TempDir()` for repository integration tests. Cover each endpoint's happy path and at least one failure path, then verify with `go test -race -cover ./...`.
- Final polish includes server timeouts, request logging (method, path, status, duration), graceful SIGINT/SIGTERM shutdown using a fresh timeout context, environment-configurable listen/database paths with local defaults, SQLite files in `.gitignore`, and a runnable README.
