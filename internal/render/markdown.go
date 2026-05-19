// Package render turns episodes into Markdown via a Go text/template.
// A built-in default template is embedded; callers may supply their own
// via Options.TemplatePath.
package render

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/lmorchard/pocketcasts-to-markdown/internal/templates"
)

// Episode is the view model handed to the template. Formatted fields are
// pre-computed so templates can stay simple (no funcs required).
type Episode struct {
	UUID                string
	Title               string
	URL                 string
	PodcastUUID         string
	PodcastTitle        string
	PodcastURL          string
	Published           string // raw ISO from the API
	PublishedFormatted  string // YYYY-MM-DD in local time, or "" on parse failure
	PlayedUpTo          int    // seconds
	PlayedUpToFormatted string // mm:ss or h:mm:ss
	Duration            int    // seconds
	DurationFormatted   string // mm:ss / h:mm:ss / "" when unknown
}

// Data is the root object the template sees.
type Data struct {
	Generated string // YYYY-MM-DD of generation, local time
	History   []Episode
	Starred   []Episode
}

// Options control which template is used and what metadata the renderer
// fills in.
type Options struct {
	// TemplatePath is the path to a custom template file. Empty means
	// use the built-in default.
	TemplatePath string
	// Now overrides the timestamp used in .Generated, primarily so tests
	// can assert deterministic output. Zero value means time.Now().
	Now time.Time
}

// Render executes the template against history+starred and returns the
// generated Markdown.
func Render(history, starred []Episode, opts Options) (string, error) {
	tplSrc, err := loadTemplate(opts.TemplatePath)
	if err != nil {
		return "", err
	}
	tpl, err := template.New("markdown").Parse(tplSrc)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	data := Data{
		Generated: now.Local().Format("2006-01-02"),
		History:   history,
		Starred:   starred,
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

// FromAPI constructs a render.Episode from the four core API/DB fields.
// Callers pass everything as plain values to keep this package free of
// imports from internal/database and internal/pocketcasts.
func FromAPI(
	uuid, title, url, podcastUUID, podcastTitle, published string,
	playedUpTo, duration int,
) Episode {
	return Episode{
		UUID:                uuid,
		Title:               title,
		URL:                 url,
		PodcastUUID:         podcastUUID,
		PodcastTitle:        podcastTitle,
		PodcastURL:          podcastURL(podcastUUID),
		Published:           published,
		PublishedFormatted:  FormatDate(published),
		PlayedUpTo:          playedUpTo,
		PlayedUpToFormatted: FormatDuration(playedUpTo),
		Duration:            duration,
		DurationFormatted:   FormatDuration(duration),
	}
}

// FormatDuration converts seconds into mm:ss (or h:mm:ss when >= 1h).
// Negative or zero returns "".
func FormatDuration(seconds int) string {
	if seconds <= 0 {
		return ""
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// FormatDate parses an ISO8601 timestamp and returns YYYY-MM-DD in local
// time. An unparseable or empty input returns "".
func FormatDate(iso string) string {
	if iso == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return ""
	}
	return t.Local().Format("2006-01-02")
}

func podcastURL(uuid string) string {
	if uuid == "" {
		return ""
	}
	return "https://pocketcasts.com/podcast/" + uuid
}

// loadTemplate returns the contents of path, or the embedded default
// when path is empty.
func loadTemplate(path string) (string, error) {
	if path == "" {
		return templates.GetDefaultTemplate()
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", path, err)
	}
	return string(b), nil
}
