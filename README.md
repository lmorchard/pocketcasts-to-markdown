# pocketcasts-to-markdown

A Go CLI that fetches your recent Pocket Casts activity (listening history
and starred episodes) into a local SQLite archive and renders Markdown
summaries from it over arbitrary date ranges.

Designed to be run unattended from cron — credentials only need to be
supplied once via the `login` command; subsequent runs use a cached
auth token.

> **Heads-up:** this uses the unofficial Pocket Casts web API at
> `api.pocketcasts.com` (the one the web player talks to). It isn't
> publicly documented and could break without notice.

## What it does

- **`login`** — exchanges your email/password for a bearer token and
  caches it locally. Run once; rerun if/when the token expires.
- **`sync`** — fetches `/user/history` and `/user/starred`, upserts every
  episode into a local SQLite DB. Idempotent; safe to run on a schedule.
  Episodes are never deleted on un-star or history rollover, so the DB
  grows over time as a personal archive.
- **`render`** — emits a Markdown document from the local archive,
  filtered by date range. No network calls. Uses an embedded default
  template; a custom template can be supplied with `--template`.

## Install

### Pre-built binaries

Tagged releases publish binaries for linux/amd64, linux/arm64,
darwin/amd64, darwin/arm64, and windows/amd64 — see the GitHub
[Releases page](https://github.com/lmorchard/pocketcasts-to-markdown/releases).
Untar (or unzip on Windows), drop the binary somewhere on your `$PATH`,
and you're done.

### Build from source

Requires Go 1.21+ and a working C toolchain (SQLite uses CGO).

```sh
git clone https://github.com/lmorchard/pocketcasts-to-markdown.git
cd pocketcasts-to-markdown
make build           # produces ./pocketcasts-to-markdown
# or:
go install ./...
```

## Quick start

```sh
# One-time login. --password-stdin avoids leaking the password into
# shell history; pipe it in from your password manager.
echo "$POCKETCASTS_PASSWORD" | pocketcasts-to-markdown login \
    --email you@example.com --password-stdin

# Pull recent activity into the local archive.
pocketcasts-to-markdown sync

# Emit a Markdown report for the last 7 days to stdout.
pocketcasts-to-markdown render --since 168h
```

## Configuration

Configuration is read from, in order of precedence:

1. Command-line flags
2. Environment variables (`POCKETCASTS_*`)
3. A YAML config file

The config file is searched at, in this order:

- `$XDG_CONFIG_HOME/pocketcasts-to-markdown/pocketcasts-to-markdown.yaml`
  (i.e. `~/.config/pocketcasts-to-markdown/...` on most Linux)
- `./pocketcasts-to-markdown.yaml` (current directory)
- Or an explicit `--config <path>`.

Recognized keys (all optional):

```yaml
# Credentials. Prefer passing these via env vars or the login command
# rather than committing them to a file.
email: ""
password: ""

# Path to the local SQLite archive.
# Default: $XDG_STATE_HOME/pocketcasts-to-markdown/state.db
# database: "/custom/path/state.db"

# Logging.
verbose: false
debug: false
log_json: false
```

Run `pocketcasts-to-markdown init` to drop a starter config and a
customizable Markdown template into the current directory.

## Commands

### `export`

```text
pocketcasts-to-markdown export --since <date|duration> [--until <date>] [-o <file>]
```

Orchestrator-friendly composition of `sync` + `render` over a single time
window. The flag shape matches the contract used by
[`me-to-markdown`](https://github.com/lmorchard/me-to-markdown) and the
rest of the `*-to-markdown` family. Section selection, item limits, and
template options are read from the config file or environment.

```bash
pocketcasts-to-markdown export --since 168h
pocketcasts-to-markdown export --since 2026-05-11 --until 2026-05-18 -o pc.md
```

`--since` is required and accepts a Go duration (`168h`) or `YYYY-MM-DD`
date. `--until` accepts `YYYY-MM-DD` and is treated as end-of-day
inclusive.

### `login`

```text
pocketcasts-to-markdown login [--email EMAIL] [--password PASS | --password-stdin]
```

Authenticates with Pocket Casts and caches the bearer token in the local
archive. Credential precedence: flags > env > config > stdin.

- `--email` (or `POCKETCASTS_EMAIL`) — account email.
- `--password` (or `POCKETCASTS_PASSWORD`) — account password. Shown in
  shell history if used this way; prefer `--password-stdin`.
- `--password-stdin` — read the password from one line of STDIN. Pairs
  well with `pass`, `gopass`, `op`, etc.

The token persists until the server invalidates it. Re-run `login` if a
subsequent `sync` complains that the token expired.

### `sync`

```text
pocketcasts-to-markdown sync
```

Fetches `/user/history` and `/user/starred`, upserts every episode into
the local archive. The DB grows monotonically — episodes don't get
removed when they fall out of the API's recent window or get un-starred.

If the cached token returns 401 and credentials are configured (env or
file), `sync` will re-login and persist the new token automatically. If
no credentials are available, it exits with an error telling you to run
`login`.

### `render`

```text
pocketcasts-to-markdown render [flags]
```

Emits a Markdown document from the local archive. No network access.

| Flag | Default | Description |
|---|---|---|
| `--since` | `168h` | Lower bound: Go duration (`168h`, `24h`) or date (`YYYY-MM-DD`). |
| `--until` | _(none)_ | Upper bound date (`YYYY-MM-DD`), inclusive. |
| `--limit` | `0` | Cap items per section. `0` = unlimited. |
| `--include` | `history,starred` | Comma-separated sections. |
| `-o`, `--output` | _(stdout)_ | Write to file instead of stdout. |
| `--template` | _(built-in)_ | Path to a custom Markdown template. |

The date-range filter applies to `published` (the episode's publish
date), not the date you listened. The starred section is not date-filtered.

### `init`

```text
pocketcasts-to-markdown init [--force] [--template-file NAME.md]
```

Creates `pocketcasts-to-markdown.yaml` and `pocketcasts-to-markdown.md`
(a copy of the embedded default template) in the current directory.
Useful for bootstrapping a new setup or as a starting point for a
custom template.

### `version`

```text
pocketcasts-to-markdown version
```

Prints version, commit, and build date.

## Custom templates

Templates are standard Go [`text/template`](https://pkg.go.dev/text/template).
`render` exposes a single root object with these fields:

| Field | Type | Notes |
|---|---|---|
| `.Generated` | string | `YYYY-MM-DD` of the render time, local. |
| `.History` | `[]Episode` | Listening history matching the date filter. |
| `.Starred` | `[]Episode` | Starred episodes. |

Each `Episode` has:

| Field | Type | Notes |
|---|---|---|
| `.UUID` | string | |
| `.Title` | string | |
| `.URL` | string | Audio URL. May be empty. |
| `.PodcastUUID` | string | |
| `.PodcastTitle` | string | May be empty for some feeds (e.g. NPR News). |
| `.PodcastURL` | string | `https://pocketcasts.com/podcast/{uuid}`. |
| `.Published` | string | Raw ISO8601 from the API. |
| `.PublishedFormatted` | string | `YYYY-MM-DD` local, or `""` if unparseable. |
| `.PlayedUpTo` | int | Seconds. |
| `.PlayedUpToFormatted` | string | `mm:ss` or `h:mm:ss`, or `""` if zero. |
| `.Duration` | int | Seconds. May be zero. |
| `.DurationFormatted` | string | `mm:ss` / `h:mm:ss` / `""`. |

See `internal/templates/default.md` for the built-in template.

## Running on a schedule

A typical cron line, assuming the binary is on `$PATH` and the token is
already cached:

```cron
# Sync hourly, regenerate the last-7-days report at 6am.
@hourly  /usr/local/bin/pocketcasts-to-markdown sync >/dev/null 2>&1
0 6 * * * /usr/local/bin/pocketcasts-to-markdown render --since 168h --output /var/www/pocketcasts.md
```

For systemd timers or other schedulers, the binary is a normal
non-interactive process — it exits 0 on success, non-zero on failure,
and logs via stderr.

## Storage

The local archive lives at, by default:

```
$XDG_STATE_HOME/pocketcasts-to-markdown/state.db
```

(falling back to `~/.local/state/pocketcasts-to-markdown/state.db` if
`XDG_STATE_HOME` is unset). Override with `--database <path>`.

Two tables of note:

- `episodes` — one row per Pocket Casts episode, keyed by UUID. Section
  flags (`in_history`, `is_starred`) only transition `0 → 1` (the upsert
  uses `MAX(...)`), so re-syncing after an un-star keeps the row marked
  as ever-starred. `first_seen_at` / `last_seen_at` track when the row
  was first observed and last refreshed.
- `kv` — the cached auth token (`auth.token`) and the last successful
  sync timestamp (`sync.last_run_at`).

## Limitations

- **The local archive can only grow forward in time.** Episodes that
  predate your first `sync` aren't recoverable — the API only returns
  the recent window.
- **Unofficial API.** Pocket Casts can change or remove endpoints at any
  time. If `sync` starts failing with parse errors, the JSON shape may
  have shifted.
- **Token expiry isn't predictable.** When a cached token starts failing
  with 401 and you haven't configured credentials, re-run `login`.

## Development

```sh
make setup    # install gofumpt + golangci-lint
make build    # produces ./pocketcasts-to-markdown
make test     # go test -race ./...
make lint     # golangci-lint
make format   # go fmt + gofumpt
```

Project layout:

```
cmd/                      Cobra subcommands + flag wiring
internal/
  config/                 Plain configuration struct
  database/               SQLite layer: schema, upserts, queries, kv
  pocketcasts/            Unofficial API client (login, history, starred)
  render/                 Markdown renderer (text/template, pre-formatted view model)
  templates/              Embedded default.md template
docs/dev-sessions/        Per-session spec/plan/notes
```

