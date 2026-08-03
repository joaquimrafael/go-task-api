# Repository Guidelines

## Project Structure & Module Organization

This repository contains a small Go REST API. The executable entry point is `cmd/api/main.go`; keep startup, routing, and dependency wiring there. HTTP handlers live under `internal/handler/`, which prevents them from being imported by unrelated modules. Add future domain logic in focused packages below `internal/` (for example, `internal/task/` or `internal/storage/`).

Place tests beside the code they exercise, using Go's `_test.go` suffix. The repository currently has no static assets or external configuration. Avoid committing generated binaries or local editor metadata.

## Build, Test, and Development Commands

- `go run ./cmd/api` starts the API locally on port `8080`.
- `curl http://localhost:8080/hello` checks the current endpoint.
- `go build ./...` compiles every package and catches build errors.
- `go test ./...` runs all repository tests.
- `go test -cover ./...` reports package-level test coverage.
- `go fmt ./...` applies standard Go formatting before review.
- `go vet ./...` checks for suspicious constructs not caught by compilation.

## Coding Style & Naming Conventions

Follow idiomatic Go and let `gofmt` determine tabs, spacing, and import grouping. Package names should be short, lowercase, and singular. Exported identifiers use `PascalCase`; unexported identifiers use `camelCase`. Name handlers by purpose, such as `CreateTask` or `ListTasks`, and keep each function focused. Handle errors explicitly and add context when returning or logging them.

## Testing Guidelines

Use Go's standard `testing` package and `net/http/httptest` for handlers. Name tests `TestFunctionName` and prefer table-driven cases when validating multiple inputs or status codes. Cover success paths, malformed requests, unsupported methods, and storage failures. No coverage threshold is currently enforced; prioritize meaningful behavior over a percentage target.

## Commit & Pull Request Guidelines

Git history is not available in this working directory, so no existing convention can be inferred. Use short, imperative commits such as `add task creation handler`. Keep commits focused. Pull requests should explain the behavior changed, list verification commands, and link any related issue. Include example requests and responses when an API contract changes; screenshots are generally unnecessary.

## Learning-Focused Contributions

This is a learning project. When assisting, explain the relevant concept, break work into small steps, and let the learner implement meaningful changes. Prefer targeted hints and code review over replacing whole files unless explicitly requested.
