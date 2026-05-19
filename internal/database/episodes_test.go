package database

import (
	"testing"
	"time"
)

func TestUpsertEpisode_RequiresUUID(t *testing.T) {
	db := newTestDB(t)
	if err := db.UpsertEpisode(Episode{}, SourceHistory); err == nil {
		t.Fatal("expected error for empty uuid")
	}
}

func TestUpsertEpisode_FlagsAreSticky(t *testing.T) {
	db := newTestDB(t)

	ep := Episode{
		UUID:         "ep-1",
		PodcastUUID:  "pod-1",
		PodcastTitle: "Pod",
		Title:        "Ep One",
		URL:          "https://example.com/1.mp3",
		Published:    "2026-05-10T00:00:00Z",
		PlayedUpTo:   42,
	}

	// First: history only.
	if err := db.UpsertEpisode(ep, SourceHistory); err != nil {
		t.Fatalf("upsert history: %v", err)
	}
	got := mustHistory(t, db, 0)
	if len(got) != 1 || got[0].UUID != "ep-1" || !got[0].InHistory || got[0].IsStarred {
		t.Fatalf("after history upsert, got %+v", got)
	}

	// Same UUID via starred — both flags should now be set.
	if err := db.UpsertEpisode(ep, SourceStarred); err != nil {
		t.Fatalf("upsert starred: %v", err)
	}
	starred := mustStarred(t, db, 0)
	if len(starred) != 1 || !starred[0].IsStarred || !starred[0].InHistory {
		t.Fatalf("after starred upsert, flags didn't merge: %+v", starred)
	}

	// History-only upsert again — starred must NOT revert to 0.
	if err := db.UpsertEpisode(ep, SourceHistory); err != nil {
		t.Fatalf("re-upsert history: %v", err)
	}
	after := mustStarred(t, db, 0)
	if len(after) != 1 || !after[0].IsStarred {
		t.Fatalf("starred flag reverted: %+v", after)
	}
}

func TestUpsertEpisode_UpdatesMutableFields(t *testing.T) {
	db := newTestDB(t)

	ep := Episode{UUID: "ep-1", Title: "old", PlayedUpTo: 10}
	if err := db.UpsertEpisode(ep, SourceHistory); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	ep.Title = "new"
	ep.PlayedUpTo = 99
	if err := db.UpsertEpisode(ep, SourceHistory); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	rows := mustHistory(t, db, 0)
	if len(rows) != 1 || rows[0].Title != "new" || rows[0].PlayedUpTo != 99 {
		t.Fatalf("expected updated fields, got %+v", rows[0])
	}
}

func TestHistoryEpisodes_FilterByPublishedRange(t *testing.T) {
	db := newTestDB(t)

	episodes := []Episode{
		{UUID: "old", Title: "old", Published: "2025-01-01T00:00:00Z"},
		{UUID: "mid", Title: "mid", Published: "2026-04-15T00:00:00Z"},
		{UUID: "new", Title: "new", Published: "2026-05-15T00:00:00Z"},
		{UUID: "no-date", Title: "no-date"}, // NULL published — excluded by range filter
	}
	for _, e := range episodes {
		if err := db.UpsertEpisode(e, SourceHistory); err != nil {
			t.Fatalf("upsert %s: %v", e.UUID, err)
		}
	}

	since := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	got, err := db.HistoryEpisodes(&since, &until, 0)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 1 || got[0].UUID != "mid" {
		t.Fatalf("expected only 'mid', got %+v", got)
	}
}

func TestHistoryEpisodes_LimitAndOrder(t *testing.T) {
	db := newTestDB(t)

	// Insert in non-chronological order to make sure ORDER BY works.
	for _, e := range []Episode{
		{UUID: "a", Published: "2026-05-01T00:00:00Z"},
		{UUID: "c", Published: "2026-05-03T00:00:00Z"},
		{UUID: "b", Published: "2026-05-02T00:00:00Z"},
	} {
		if err := db.UpsertEpisode(e, SourceHistory); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}

	got, err := db.HistoryEpisodes(nil, nil, 2)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected limit=2, got %d", len(got))
	}
	if got[0].UUID != "c" || got[1].UUID != "b" {
		t.Fatalf("expected DESC order c,b — got %s,%s", got[0].UUID, got[1].UUID)
	}
}

func TestStarredEpisodes_OnlyReturnsStarred(t *testing.T) {
	db := newTestDB(t)

	if err := db.UpsertEpisode(Episode{UUID: "h"}, SourceHistory); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertEpisode(Episode{UUID: "s"}, SourceStarred); err != nil {
		t.Fatal(err)
	}

	got, err := db.StarredEpisodes(0)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 1 || got[0].UUID != "s" {
		t.Fatalf("expected only starred 's', got %+v", got)
	}
}

func mustHistory(t *testing.T, db *DB, limit int) []Episode {
	t.Helper()
	got, err := db.HistoryEpisodes(nil, nil, limit)
	if err != nil {
		t.Fatalf("HistoryEpisodes: %v", err)
	}
	return got
}

func mustStarred(t *testing.T, db *DB, limit int) []Episode {
	t.Helper()
	got, err := db.StarredEpisodes(limit)
	if err != nil {
		t.Fatalf("StarredEpisodes: %v", err)
	}
	return got
}
