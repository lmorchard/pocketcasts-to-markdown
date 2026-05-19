# Notes

## Session start — 2026-05-18 20:01

Greenfield project. Just `.git` in the dir, no commits yet.

Les pointed me at his existing JS client: `lmorchard/about-me/lib/pocketcasts-client.js`. Key things confirmed from that + `cards/PocketCasts/fetch.js`:

- API base: `https://api.pocketcasts.com`
- Login: `POST /user/login` with `{ email, password, scope: "webplayer" }` → `{ token, ... }`
- All authenticated reads are POSTs with `Authorization: Bearer <token>`.
- History response shape: `{ episodes: [...] }`, where each episode has `uuid`, `url`, `title`, `podcastTitle`, `podcastUuid`, `published`, `playedUpTo`.
- Starred endpoint is unverified — assuming `user/starred` with similar shape; will check on first live run.

Les's answers to the kickoff questions:
- Data source: unofficial web API w/ login (referenced the JS client).
- Scope: played/completed + starred (not Up Next or new subscriptions).
- Process: dev-session scaffolding first.

## Decisions after first review

- **SQLite is in.** Les wants date-range queries that work beyond whatever window the live API returns. Local DB acts as a growing archive; render reads from DB, not from live API.
- **Token caching is in.** Will be run unattended. Cache token in `kv` table; re-login on 401 with one retry.
- **Module path:** `github.com/lmorchard/pocketcasts-to-markdown`.
- **Binary name:** `pocketcasts-to-markdown` (no short alias for v1).
- **Starred endpoint:** unknown. Discover at runtime in Phase 4.

## Running log

### 2026-05-18 ~20:25 — Phase 2 storage layer complete

- Extended the scaffold's `internal/database` package rather than a new `internal/store` — keeps the package surface consistent with the skill's conventions and means migrations + upserts live next to each other.
- **Sticky flag merge:** `ON CONFLICT(uuid) DO UPDATE ... is_starred = MAX(is_starred, excluded.is_starred)` — flags only go 0→1, never the reverse. Matches the "growing archive" decision from the spec.
- **Nullable fields:** scan columns through `COALESCE(col, '')` / `COALESCE(col, 0)` so the Go struct stays plain `string` / `int` rather than `sql.NullString` everywhere.
- **Date filtering:** `published` stays a string column; we format `time.Time` to RFC3339 in queries and let SQLite do lexicographic comparison. Episodes with NULL `published` are excluded by range filters (intentional — they have no date to filter against).
- 9 tests across upsert idempotency, flag stickiness, mutable-field updates, range filter, limit + ordering, starred-only filter, and full kv set/get/delete. All green under `-race`.

### 2026-05-18 ~20:10 — Phase 1 scaffolding complete

- Ran `scaffold_project.py pocketcasts-to-markdown --templates` (into a tmp dir; copied into the existing project dir which had `.git/` and `docs/`).
- **Skill bug noticed:** `scaffold_project.py` only substitutes `{{KEY}}`-style markers, but `init.go.template` and `default.md.template` use Go-template `{{.X}}` syntax — those files come out with unrendered placeholders. Rewrote both files locally; logging this for a future PR back to the skill repo.
- **Driver decision:** went with the skill's default `mattn/go-sqlite3` (CGO) rather than swapping to `modernc.org/sqlite`. Aligns with the Makefile/CI the skill ships.
- **Path conventions:** moved DB and config defaults to XDG paths (`$XDG_STATE_HOME/pocketcasts-to-markdown/state.db`, `$XDG_CONFIG_HOME/pocketcasts-to-markdown/pocketcasts-to-markdown.yaml`). The skill defaulted to cwd, but unattended cron use makes XDG the right default.
- **Env binding:** `viper.SetEnvPrefix("POCKETCASTS")` + `AutomaticEnv()` so `viper.Get("email")` → `POCKETCASTS_EMAIL`.
- **Schema:** `episodes` (with `is_starred` + `in_history` flags, first/last_seen_at) and `kv` (for cached auth token and sync metadata).
- Builds, vets, tests, and lints clean. `init` smoke-tested in a temp dir.
- Two errcheck lint hits came from the skill's `database.go.template` — fixed locally with `_ = conn.Close()`. Probably worth upstreaming.
