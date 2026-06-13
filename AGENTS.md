# AGENTS.md

## Project Overview

LazyDB is a Go terminal UI database browser and editor. It supports SQLite, MySQL/MariaDB, and PostgreSQL.

The app uses Bubble Tea for the TUI, Lip Gloss for styling, and `database/sql` with database-specific drivers for persistence.

## Repository Layout

- `main.go`: application entry point.
- `internal/tui/`: Bubble Tea model, update loop, rendering, tables, forms, overlays, and keyboard handling.
- `internal/db/`: database connection logic, query templates, row scanning, CRUD operations, and supported database constants.
- `internal/utils/`: shared utilities such as file logging.
- `README.md`: installation and feature overview.
- `SPEC.md`: local feature notes, currently ignored by git.
- `PROMPT.md`: local planning prompt, not general project documentation.

## Common Commands

- Run the app: `go run .`
- Build all packages: `go build ./...`
- Run tests: `go test ./...`
- Install CLI locally: `go install .`

## Development Notes

- Keep database-specific SQL templates in `internal/db/queries.go`.
- Keep supported database constants and defaults in `internal/db/defaults.go`.
- Prefer adding database behavior through `QueryMap`, `GetQuery`, `Placeholder`, and `quoteIdent` rather than scattering SQL strings.
- TUI screen state is managed by the `screen` enum and `Model.stack` in `internal/tui/app.go`.
- Row viewing, inserts, updates, and deletes depend on primary key discovery. Handle missing primary keys gracefully.
- The app logs to `app.log`, which is ignored by git.
- Generated binaries, local databases, scratch files, and `AGENTS.md` are ignored by git.

## Code Style

- Use `gofmt` on Go changes.
- Keep changes small and localized.
- Prefer existing patterns over introducing new abstractions.
- Use parameter placeholders for user values. Avoid string-concatenating untrusted values into SQL.
- Identifier quoting is centralized through `quoteIdent`; use it for table and column names.

## Verification

Before finishing code changes, run:

```bash
go test ./...
go build ./...
```

If changes affect database behavior, manually verify the relevant flow with at least one supported database when feasible.
