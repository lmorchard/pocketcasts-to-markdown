package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Source indicates which Pocket Casts endpoint an episode came from
// during a sync. UpsertEpisode uses this to set the matching flag
// without clobbering the other one (flags are sticky: 0 -> 1 only).
type Source int

const (
	SourceHistory Source = iota
	SourceStarred
)

// Episode is the in-memory representation of a row in the episodes table.
// String fields default to "" when the underlying column is NULL.
type Episode struct {
	UUID         string
	PodcastUUID  string
	PodcastTitle string
	Title        string
	URL          string
	Published    string // ISO8601 as returned by the API
	PlayedUpTo   int    // seconds
	Duration     int    // seconds, 0 if unknown
	IsStarred    bool
	InHistory    bool
	FirstSeenAt  time.Time
	LastSeenAt   time.Time
}

const upsertEpisodeSQL = `
INSERT INTO episodes (
    uuid, podcast_uuid, podcast_title, title, url, published,
    played_up_to, duration, is_starred, in_history,
    first_seen_at, last_seen_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(uuid) DO UPDATE SET
    podcast_uuid  = excluded.podcast_uuid,
    podcast_title = excluded.podcast_title,
    title         = excluded.title,
    url           = excluded.url,
    published     = excluded.published,
    played_up_to  = excluded.played_up_to,
    duration      = excluded.duration,
    is_starred    = MAX(is_starred, excluded.is_starred),
    in_history    = MAX(in_history, excluded.in_history),
    last_seen_at  = CURRENT_TIMESTAMP
`

// UpsertEpisode inserts ep or merges it into the existing row keyed by UUID.
// The matching section flag (is_starred or in_history) is set to 1 based on
// src; the other flag is preserved (we use MAX so a 1 never reverts to 0).
//
// Returns an error if ep.UUID is empty.
func (db *DB) UpsertEpisode(ep Episode, src Source) error {
	if ep.UUID == "" {
		return errors.New("episode uuid is required")
	}

	starred, history := 0, 0
	switch src {
	case SourceHistory:
		history = 1
	case SourceStarred:
		starred = 1
	default:
		return fmt.Errorf("unknown source: %d", src)
	}

	_, err := db.conn.Exec(
		upsertEpisodeSQL,
		ep.UUID, ep.PodcastUUID, ep.PodcastTitle, ep.Title, ep.URL,
		nullableString(ep.Published), ep.PlayedUpTo, ep.Duration,
		starred, history,
	)
	if err != nil {
		return fmt.Errorf("upsert episode %s: %w", ep.UUID, err)
	}
	return nil
}

const selectEpisodeColumns = `
    uuid,
    COALESCE(podcast_uuid, ''),
    COALESCE(podcast_title, ''),
    COALESCE(title, ''),
    COALESCE(url, ''),
    COALESCE(published, ''),
    COALESCE(played_up_to, 0),
    COALESCE(duration, 0),
    is_starred,
    in_history,
    first_seen_at,
    last_seen_at
`

// HistoryEpisodes returns episodes flagged as in_history, optionally filtered
// by published date range. since/until may be nil for no bound. Results are
// ordered by published DESC, then last_seen_at DESC.
//
// limit <= 0 means unlimited.
func (db *DB) HistoryEpisodes(since, until *time.Time, limit int) ([]Episode, error) {
	return db.queryEpisodes("in_history = 1", since, until, limit)
}

// StarredEpisodes returns episodes flagged as is_starred. Ordered by
// last_seen_at DESC. limit <= 0 means unlimited.
func (db *DB) StarredEpisodes(limit int) ([]Episode, error) {
	return db.queryEpisodes("is_starred = 1", nil, nil, limit)
}

func (db *DB) queryEpisodes(flagClause string, since, until *time.Time, limit int) ([]Episode, error) {
	var (
		conds = []string{flagClause}
		args  []any
	)
	if since != nil {
		conds = append(conds, "published >= ?")
		args = append(args, since.UTC().Format(time.RFC3339))
	}
	if until != nil {
		conds = append(conds, "published <= ?")
		args = append(args, until.UTC().Format(time.RFC3339))
	}

	q := "SELECT " + selectEpisodeColumns + " FROM episodes WHERE " +
		strings.Join(conds, " AND ") +
		" ORDER BY published DESC, last_seen_at DESC"
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.conn.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("query episodes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Episode
	for rows.Next() {
		var ep Episode
		var isStarred, inHistory int
		if err := rows.Scan(
			&ep.UUID, &ep.PodcastUUID, &ep.PodcastTitle, &ep.Title, &ep.URL,
			&ep.Published, &ep.PlayedUpTo, &ep.Duration,
			&isStarred, &inHistory, &ep.FirstSeenAt, &ep.LastSeenAt,
		); err != nil {
			return nil, fmt.Errorf("scan episode: %w", err)
		}
		ep.IsStarred = isStarred != 0
		ep.InHistory = inHistory != 0
		out = append(out, ep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate episodes: %w", err)
	}
	return out, nil
}

// nullableString returns sql.NullString so empty values land as SQL NULL.
// Keeps queries that filter by date able to use IS NULL / NOT NULL.
func nullableString(s string) any {
	if s == "" {
		return sql.NullString{}
	}
	return s
}
