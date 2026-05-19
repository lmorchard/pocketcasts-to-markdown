package render

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		seconds int
		want    string
	}{
		{0, ""},
		{-5, ""},
		{42, "0:42"},
		{60, "1:00"},
		{125, "2:05"},
		{3600, "1:00:00"},
		{3661, "1:01:01"},
		{36000 + 5*60 + 9, "10:05:09"},
	}
	for _, c := range cases {
		if got := FormatDuration(c.seconds); got != c.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", c.seconds, got, c.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"not a date", ""},
		{"2026-05-15T12:34:56Z", "2026-05-15"}, // UTC may shift in local TZ; checked below
	}
	for _, c := range cases[:2] {
		if got := FormatDate(c.in); got != c.want {
			t.Errorf("FormatDate(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	// Valid input: just confirm format is YYYY-MM-DD (10 chars, dashes in place).
	got := FormatDate("2026-05-15T12:34:56Z")
	if len(got) != 10 || got[4] != '-' || got[7] != '-' {
		t.Errorf("FormatDate ISO produced unexpected format: %q", got)
	}
}

func TestFromAPI_PopulatesFormatted(t *testing.T) {
	ep := FromAPI("ep-1", "Title", "https://x", "pod-1", "Pod", "2026-05-15T12:00:00Z", 125, 3600)
	if ep.PlayedUpToFormatted != "2:05" {
		t.Errorf("PlayedUpToFormatted = %q, want 2:05", ep.PlayedUpToFormatted)
	}
	if ep.DurationFormatted != "1:00:00" {
		t.Errorf("DurationFormatted = %q, want 1:00:00", ep.DurationFormatted)
	}
	if ep.PodcastURL != "https://pocketcasts.com/podcast/pod-1" {
		t.Errorf("PodcastURL = %q", ep.PodcastURL)
	}
	if ep.PublishedFormatted == "" {
		t.Errorf("PublishedFormatted should be non-empty for valid date")
	}
}

func TestFromAPI_HandlesEmptyFields(t *testing.T) {
	ep := FromAPI("ep-1", "Title", "", "", "", "", 0, 0)
	if ep.PlayedUpToFormatted != "" || ep.DurationFormatted != "" {
		t.Errorf("expected empty formatted durations, got %+v", ep)
	}
	if ep.PodcastURL != "" {
		t.Errorf("expected empty PodcastURL for missing UUID, got %q", ep.PodcastURL)
	}
	if ep.PublishedFormatted != "" {
		t.Errorf("expected empty PublishedFormatted, got %q", ep.PublishedFormatted)
	}
}

func TestRender_EmptyCollectionsProducesHeaderOnly(t *testing.T) {
	out, err := Render(nil, nil, Options{Now: fixedTime()})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "# Pocket Casts — Recent Activity") {
		t.Errorf("missing main header in output: %q", out)
	}
	if strings.Contains(out, "## Listening history") || strings.Contains(out, "## Starred") {
		t.Errorf("empty collections should suppress section headers: %q", out)
	}
}

func TestRender_HistoryAndStarred(t *testing.T) {
	history := []Episode{
		FromAPI("ep-1", "Ep One", "https://example.com/1.mp3", "pod-1", "My Pod", "2026-05-15T00:00:00Z", 125, 0),
	}
	starred := []Episode{
		FromAPI("ep-2", "Starred Ep", "", "pod-2", "Other Pod", "2026-05-10T00:00:00Z", 0, 0),
	}
	out, err := Render(history, starred, Options{Now: fixedTime()})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	wantSubstrings := []string{
		"## Listening history",
		"[Ep One](https://example.com/1.mp3)",
		"[My Pod](https://pocketcasts.com/podcast/pod-1)",
		"played 2:05",
		"## Starred",
		"**Starred Ep**", // no URL — falls back to plain bold
		"[Other Pod](https://pocketcasts.com/podcast/pod-2)",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q\n---\n%s\n---", s, out)
		}
	}
}

func TestRender_EmptyPodcastTitleSuppressesLink(t *testing.T) {
	// NPR News-style: title but no podcast_title. Should not emit "[]()".
	history := []Episode{
		FromAPI("ep-1", "NPR News 1PM", "https://example.com/1.mp3", "", "", "2026-05-15T00:00:00Z", 30, 0),
	}
	out, err := Render(history, nil, Options{Now: fixedTime()})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "[]()") {
		t.Errorf("output contains broken empty link\n---\n%s\n---", out)
	}
	if !strings.Contains(out, "NPR News 1PM") {
		t.Errorf("title missing from output: %q", out)
	}
}

func TestRender_CustomTemplate(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/tpl.md"
	if err := writeFile(path, "GENERATED:{{ .Generated }}\nN:{{ len .History }}\n"); err != nil {
		t.Fatal(err)
	}
	out, err := Render(
		[]Episode{FromAPI("ep-1", "T", "", "", "", "", 0, 0)},
		nil,
		Options{TemplatePath: path, Now: fixedTime()},
	)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "GENERATED:") || !strings.Contains(out, "N:1") {
		t.Errorf("custom template not honored: %q", out)
	}
}

func fixedTime() time.Time {
	return time.Date(2026, 5, 18, 20, 0, 0, 0, time.UTC)
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
