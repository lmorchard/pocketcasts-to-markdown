# Todo

## Phase 1 — Scaffolding
- [x] Run go-cli-builder scaffold (`--templates`)
- [x] Replace `github.com/yourusername/...` with real module path
- [x] Fix unrendered placeholders in `cmd/init.go` and `internal/templates/default.md`
- [x] Config struct + viper bindings for `POCKETCASTS_EMAIL` / `POCKETCASTS_PASSWORD`
- [x] DB path resolution (XDG_STATE_HOME)
- [x] Config file resolution (XDG_CONFIG_HOME)
- [x] Stub `sync` and `render` subcommands (with full flag set on render)
- [x] Replace empty schema.sql with episodes + kv tables
- [x] Rewrite default.md template for our actual output shape
- [x] Update `.yaml.example`
- [x] `make build` + `make test` green
- [x] `golangci-lint run` clean (after `_ = conn.Close()` fix in scaffold-generated code)
- [x] Initial commit (`bc934f4`)
- [x] Skill bugs fixed upstream in `lmorchard/lmorchard-agent-skills` (`f6ba1d1`)

## Phase 2 — Storage
- [x] Schema (already in `internal/database/schema.sql` from Phase 1)
- [x] `internal/database/episodes.go` — `Episode`, `Source`, `UpsertEpisode`, `HistoryEpisodes`, `StarredEpisodes`
- [x] `internal/database/kv.go` — `GetKV`, `SetKV`, `DeleteKV`
- [x] Upsert uses `ON CONFLICT(uuid)` with sticky flag merge (`MAX(is_starred, excluded.is_starred)`)
- [x] Unit tests against temp DB (9 tests, all green under `-race`)
- [x] `go vet ./...` clean, `golangci-lint run` clean

## Phase 3 — API client
- [x] `internal/pocketcasts/client.go` (`Client`, options pattern, `Login`, `History`, `Starred`)
- [x] `internal/pocketcasts/errors.go` (`APIError`, `IsUnauthorized`, `ErrNotAuthenticated`)
- [x] Response structs (`loginResponse`, `episodeListResponse`, `Episode`)
- [x] Typed error with HTTP status — callers can `errors.As(&apiErr)` and check `StatusCode == 401`
- [x] 8 httptest unit tests (request shape, auth header, 401 surfacing, decode errors)
- [x] Removed duplicate `APIBaseURL` from `cmd/constants.go`

## Phase 4 — `sync`
- [x] Token cache load/store via kv (key `auth.token`)
- [x] 401 → re-login → retry once (`fetchWithRelogin`, applies to both endpoints)
- [x] `/user/starred` confirmed working — no probe needed
- [x] Upsert history (`in_history` flag)
- [x] Upsert starred (`is_starred` flag)
- [x] Set `sync.last_run_at`
- [x] Summary log line
- [x] 4 unit tests for the 401/relogin paths
- [x] Live smoke test against real account: 100 history + 5 starred, second run reuses cached token

## Phase 5 — `render`
- [ ] `internal/render/markdown.go`
- [ ] Duration & date helpers
- [ ] Flag parsing for `--since` (date or duration)
- [ ] Top-level default = sync + render
- [ ] Snapshot tests

## Phase 6 — Polish
- [ ] README + cron example
- [ ] GitHub Actions release workflow
- [ ] Verify starred endpoint, document in notes
