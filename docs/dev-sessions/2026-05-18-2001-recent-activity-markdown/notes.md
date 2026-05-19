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

### 2026-05-18 ~20:52 — Added `login` subcommand

- Les flagged that keeping email+password in a config file isn't ideal. New `login` subcommand authenticates once and caches the token; subsequent `sync` runs need no credentials.
- Credential precedence for `login`: flags > env > config > `--password-stdin` (read from STDIN). The `--password-stdin` pattern follows `docker login`'s convention — secure, no shell history leak, scriptable.
- `sync` now allows running with just a cached token (no creds). A 401 with no creds returns an actionable error pointing to the `login` command rather than silently failing.
- **Dep excursion learned the hard way:** `go get golang.org/x/term@latest` transitively bumped `go.mod`'s `go` directive to `1.25.0`. CI uses Go 1.23, so that would have broken builds. Dropped x/term entirely (no-echo prompts not worth the dep churn for an unattended-cron tool); pinned `x/sys` back to v0.29.0; manually reset `go` directive to 1.21. `go.mod` / `go.sum` now identical to before the excursion.
- Live test: `echo $PASS | login --email ... --password-stdin` cached the token; subsequent `sync` (no env, no config creds) pulled 100 history + 5 starred successfully.

### 2026-05-18 ~20:48 — Phase 4 sync complete, live-verified

- `cmd/sync.go` wires the API client to the storage layer. `fetchWithRelogin(ctx, log, client, email, password, saveToken, fn)` runs `fn`, on 401 logs in fresh, persists via the injected `saveToken` callback, and retries once. Both history and starred go through this path so token expiry between calls is handled too.
- The `saveToken` callback (closing over `db.SetKV`) keeps the helper decoupled from the storage layer — easy to drive from `httptest` in unit tests.
- 4 unit tests cover no-cached-token, valid-cached, stale-cached→relogin, and login-failure.
- **Live smoke test against Les's real account:**
  - Login worked. Token cached. Second run produced `using cached auth token` (no relogin).
  - `/user/history` returned 100 episodes. JSON shape exactly matched our DTO.
  - `/user/starred` returned 5 episodes — endpoint guess was correct.
  - All 105 episodes have URLs; `duration` field IS populated (useful for render).
  - NPR News episodes have empty `podcastTitle` — renderer should handle gracefully (fall back to title only, or "Unknown podcast").
  - Second sync is idempotent: same 105 rows, no duplicates.

### 2026-05-18 ~20:35 — Phase 3 API client complete

- `internal/pocketcasts/client.go` + `errors.go`. Options pattern (`WithBaseURL`, `WithHTTPClient`) so tests can point at `httptest.NewServer`.
- Single private `do(ctx, op, path, body, authed, out)` handles marshal/header/Authorization/decode. Non-2xx → `*APIError` carrying op, status, and up to 4KiB of body for diagnostics.
- `IsUnauthorized(err)` uses `errors.As` to detect 401 specifically — Phase 4 will use this to drive the relogin-and-retry path.
- `Login` stores the token on the client AND returns it, so callers can decide whether to cache it.
- 8 httptest tests cover success path, bad creds → APIError, missing token in response, history-without-token, bearer + decode, 401 on history, starred path, malformed JSON.

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
