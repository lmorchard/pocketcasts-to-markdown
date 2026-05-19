# Plan

Build in small, testable slices. Each phase ends in a runnable binary.

## Phase 1 — Project scaffolding

- `go mod init github.com/lmorchard/pocketcasts-to-markdown`
- Cobra + Viper + `modernc.org/sqlite` (pure-Go SQLite, no CGO) per the `go-cli-builder` skill.
- `root` command with global flags: `--config`, `--db`, `--verbose`.
- Config sources (precedence): flags > env (`POCKETCASTS_*`) > config file (`~/.config/pocketcasts-to-markdown/config.yaml`).
- Makefile (`build`, `run`, `test`, `lint`, `fmt`).
- `.gitignore`.

**Exit:** `pocketcasts-to-markdown --help` lists subcommand stubs; config + db path resolution prints under `--verbose`.

## Phase 2 — Storage layer

- `internal/store/store.go` with a `Store` struct wrapping `*sql.DB`.
- Migrations applied on open (single embedded `schema.sql` for v1; switch to numbered migrations if/when we evolve).
- Methods: `UpsertEpisode`, `SetStarred`, `MarkInHistory`, `GetKV`, `SetKV`, `EpisodesBetween(since, until, filters)`.
- Unit tests against a temp-file DB.

**Exit:** `go test ./internal/store/...` passes; can manually insert + query rows.

## Phase 3 — API client

- `internal/pocketcasts/client.go` with `Login`, `History`, `Starred`.
- Typed response structs matching observed JSON.
- Bearer-token state on the client struct.
- Error type carrying HTTP status (so callers can distinguish 401).
- `httptest` unit tests for request shape, auth header, error mapping.
- Token plumbing: client takes an injected token-provider (function or interface) so the store-backed cache lives in the caller, not the client.

**Exit:** `go test ./internal/pocketcasts/...` passes.

## Phase 4 — `sync` subcommand

- Resolve config + open store.
- Token lifecycle: load cached token from `kv`; if present, try it; on 401, re-login; persist new token.
- Discover starred endpoint: try `/user/starred`; on 404, log and probe a couple of alternates (`/user/starred/list`, `/user/episodes/starred`). Document what works in `notes.md`.
- Upsert history episodes (`in_history = 1`).
- Upsert starred episodes (`is_starred = 1`).
- Record `sync.last_run_at`.
- Print a one-line summary: `synced N history, M starred (T new)`.

**Exit:** Running `pocketcasts-to-markdown sync` against Les's real account populates the DB.

## Phase 5 — `render` subcommand + default

- `internal/render/markdown.go` — pure function, takes episodes + opts, returns markdown.
- Flags: `--since`, `--until`, `--limit`, `--include`, `--output`.
- Date parsing accepts either `YYYY-MM-DD` or Go duration (`168h`).
- Default top-level run does `sync` + `render`.
- Snapshot tests with fixture episodes.

**Exit:** `pocketcasts-to-markdown render --since 7d` produces a sane markdown file.

## Phase 6 — Polish

- README with install + cron example.
- GitHub Actions release workflow per `go-cli-builder` skill.
- Verify starred endpoint and lock it in.

## Deferred

- Pagination, if the API turns out to truncate history.
- Re-sync of historical data we missed (likely impossible — note as known limitation).
- Multiple users / accounts.
