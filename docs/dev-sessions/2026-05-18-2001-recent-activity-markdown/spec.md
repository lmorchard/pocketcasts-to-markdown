# Spec: pocketcasts-to-markdown — Recent Activity → Markdown

## Goal

A Go CLI that fetches recent activity from Pocket Casts (listening history + starred episodes) and emits a Markdown document summarizing it.

## Background

- Pocket Casts has no official public API. We use the unofficial web API at `https://api.pocketcasts.com`, the same one Les's existing JS client (`about-me/lib/pocketcasts-client.js`) targets.
- Auth: `POST /user/login` with `{ email, password, scope: "webplayer" }` → returns `{ token }`. Subsequent calls send `Authorization: Bearer <token>`.
- All authenticated endpoints are POSTs, even reads.

## Endpoints we use

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/user/login` | POST | Exchange email/password for a bearer token |
| `/user/history` | POST | Listening history (recent first) |
| `/user/starred` | POST | Starred episodes (to be verified at runtime) |

## Episode response shape (from history)

Each item in `response.episodes`:

- `uuid` — episode id
- `url` — audio URL
- `title` — episode title
- `podcastTitle`
- `podcastUuid` — used to construct `https://pocketcasts.com/podcast/{uuid}` and the artwork URL
- `published` — ISO timestamp of episode publish date
- `playedUpTo` — seconds played

(Starred shape assumed similar; verify at runtime.)

## CLI behavior (v1)

- Single binary, `pocketcasts-to-markdown`.
- Designed to run unattended (cron / scheduler). No interactive prompts.
- Reads credentials from env vars or a config file:
  - `POCKETCASTS_EMAIL`
  - `POCKETCASTS_PASSWORD`
- Subcommands:
  - `sync` — login (using cached token if valid), fetch history + starred, upsert into local SQLite. Idempotent.
  - `render` — emit markdown from the local SQLite, filtered by date range. No network.
  - (Default / top-level run does `sync` then `render` for convenience.)
- Flags for `render`:
  - `--since <date|duration>` — e.g. `2026-04-01` or `168h`. Filters by `played-at`/`published`. Default 7 days.
  - `--until <date>` — optional upper bound. Default now.
  - `--limit <n>` — cap items per section. Default unlimited within the window.
  - `--include <history,starred>` — which sections to emit. Default both.
  - `--output, -o <path>` — file path. Default stdout.
- Global flags: `--config <path>`, `--db <path>`.
- Exit non-zero on auth failure or API errors.

## Markdown output (v1 shape)

```markdown
# Pocket Casts — Recent Activity

_Generated 2026-05-18_

## Listening history

- **[Episode title](episode-url)** — [Podcast title](podcast-url) · played 12:34 / 56:78 · published 2026-05-17
- ...

## Starred

- **[Episode title](episode-url)** — [Podcast title](podcast-url)
- ...
```

Details:
- Times in `mm:ss` or `h:mm:ss`.
- Dates rendered as `YYYY-MM-DD` in local time.
- Skip items with no `url` rather than emit dead links? Open question — for now, emit the title without a link.

## Persistence (SQLite)

Stored at `${XDG_STATE_HOME:-~/.local/state}/pocketcasts-to-markdown/state.db` by default, overridable with `--db`.

Schema (sketch — finalize in Phase 2):

```sql
CREATE TABLE episodes (
  uuid           TEXT PRIMARY KEY,
  podcast_uuid   TEXT,
  podcast_title  TEXT,
  title          TEXT,
  url            TEXT,
  published      TEXT,     -- ISO8601
  played_up_to   INTEGER,  -- seconds
  duration       INTEGER,  -- seconds, if API returns it
  is_starred     INTEGER NOT NULL DEFAULT 0,
  in_history     INTEGER NOT NULL DEFAULT 0,
  first_seen_at  TEXT NOT NULL,
  last_seen_at   TEXT NOT NULL
);
CREATE INDEX idx_episodes_published ON episodes(published);

CREATE TABLE kv (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
-- kv keys: "auth.token", "auth.email", "auth.expires_at" (if available), "sync.last_run_at"
```

Sync behavior:
- Upsert by `uuid`. Always update `last_seen_at` and section flags (`in_history`, `is_starred`).
- We never delete rows on un-star or history rollover — the local DB is a growing archive.
- Known limitation: history that predates the first run is unrecoverable; we only see what the API returns now and onward.

## Token caching

- Token stored in the `kv` table after a successful login.
- On startup of `sync`, if a cached token exists, try it. On `401`, re-login and retry once.
- If the login response includes an expiry, store it and treat as expired N minutes early. If not, rely on the 401-retry path.

## Non-goals (v1)

- No OPML / subscriptions / Up Next.
- No pagination handling beyond what `/user/history` returns by default (revisit if needed once we see the live shape).
- No interactive auth flow. Email + password only.

## Open questions

1. Does `/user/history` accept a date-range parameter, or do we just upsert everything it returns and let the DB filter on render? Assume the latter.
2. Confirm `/user/starred` is correct — discover at runtime (try it; if 404, probe alternates).
3. Does the login response include token expiry / TTL? Inspect on first run.
4. Should episodes without `url` be skipped, or rendered as plain text? Default: render title as plain text.
