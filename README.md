# Go Task API

A small REST API written in Go for learning how to build and organize a web
service. The project will provide endpoints for managing tasks; currently, it
includes a simple health-style greeting endpoint.

## Project structure

```text
cmd/api/           Application entry point and server setup
internal/handler/  HTTP request handlers
```

## Requirements

- Go version compatible with the version declared in `go.mod`

## Run locally

Start the API from the repository root:

```bash
go run ./cmd/api
```

The server listens at `http://localhost:8080`. In another terminal, test the
current endpoint:

```bash
curl http://localhost:8080/hello
```

Expected response:

```text
Hello World!
```

## Development checks

```bash
go fmt ./...   # Format Go source files
go test ./...  # Run all tests
go vet ./...   # Find common correctness issues
go build ./... # Verify that all packages compile
```

## Ignored files

The `.gitignore` excludes generated binaries, test and coverage output,
temporary files, local environment variables, and operating-system or editor
metadata. These files are machine-specific or reproducible and should not be
committed. Source code, `go.mod`, and `go.sum` should remain tracked so builds
are reproducible.
