# Go Task API

A small REST API for managing tasks, built with Go's standard `net/http`
package and SQLite. The project demonstrates a layered handler, service, and
repository architecture without a web framework.

## Project structure

```text
cmd/api/              Application startup, dependency wiring, and routes
internal/handler/     HTTP handlers and JSON response helpers
internal/model/       Task types and domain errors
internal/repository/  SQLite setup and persistence
internal/service/     Validation and business rules
specs/                Detailed implementation guide
```

## Requirements

- Go 1.26.4 or a compatible version declared in `go.mod`

## Run locally

Start the API from the repository root:

```bash
go run ./cmd/api
```

The server listens at `http://localhost:8080` and creates `tasks.db` in the
repository root. In another terminal, check the server and database:

```bash
curl -i http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```

## API

| Method | Path | Success | Description |
| --- | --- | --- | --- |
| `GET` | `/health` | `200` | Check database availability |
| `GET` | `/tasks` | `200` | List tasks ordered by ID |
| `GET` | `/tasks/{id}` | `200` | Retrieve one task |
| `POST` | `/tasks` | `201` | Create a task |
| `PUT` | `/tasks/{id}` | `200` | Replace a task's writable fields |
| `DELETE` | `/tasks/{id}` | `204` | Delete a task with no response body |

A task input contains `title`, `description`, and `completed`. Titles are
trimmed and must contain between 1 and 120 characters; descriptions are
optional.

Create a task:

```bash
curl -i -X POST http://localhost:8080/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Study Go","description":"Finish the API","completed":false}'
```

List or retrieve tasks:

```bash
curl -i http://localhost:8080/tasks
curl -i http://localhost:8080/tasks/1
```

Replace task `1`:

```bash
curl -i -X PUT http://localhost:8080/tasks/1 \
  -H 'Content-Type: application/json' \
  -d '{"title":"Study Go","description":"API complete","completed":true}'
```

Delete task `1`:

```bash
curl -i -X DELETE http://localhost:8080/tasks/1
```

JSON request bodies must contain exactly one object and cannot contain unknown
fields. Invalid IDs and malformed JSON return `400`, validation failures return
`422`, missing tasks return `404`, and unexpected failures return `500`. Error
responses have the form:

```json
{"error":"task not found"}
```

## Development checks

```bash
go fmt ./...   # Format Go source files
go test ./...  # Run all tests
go vet ./...   # Find common correctness issues
go build ./... # Verify that all packages compile
```

Repository integration tests use temporary SQLite databases and cover schema
initialization and repository CRUD behavior. Table-driven service and handler
tests cover validation, error mapping, context propagation, and every endpoint.
A lightweight in-process API integration test exercises the real application
stack through a complete CRUD lifecycle.

## Current limitations

The listen address and database path are currently fixed at `:8080` and
`tasks.db`. Environment-based configuration and generated SQLite-file ignore
rules remain to be added in the final phase of the implementation guide.
