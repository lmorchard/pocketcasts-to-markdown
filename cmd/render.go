package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lmorchard/pocketcasts-to-markdown/internal/database"
	"github.com/lmorchard/pocketcasts-to-markdown/internal/render"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// renderCmd reads episodes from the local archive and emits Markdown.
var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render Markdown from the local archive",
	Long: `Read episodes from the local SQLite archive and emit a Markdown
document summarizing recent activity. No network calls.

--since accepts either a Go duration ("168h", "7d-style values are not
supported by time.ParseDuration; use 168h for 7 days") or a date in
YYYY-MM-DD form.`,
	RunE: runRender,
}

func init() {
	rootCmd.AddCommand(renderCmd)

	renderCmd.Flags().String("since", "168h", "filter window — Go duration (e.g. 168h) or date (YYYY-MM-DD)")
	renderCmd.Flags().String("until", "", "upper bound date (YYYY-MM-DD), default now")
	renderCmd.Flags().Int("limit", 0, "max items per section (0 = unlimited)")
	renderCmd.Flags().String("include", "history,starred", "comma-separated sections to emit")
	renderCmd.Flags().StringP("output", "o", "", "output file path (default: stdout)")
	renderCmd.Flags().String("template", "", "custom markdown template file (default: built-in)")

	_ = viper.BindPFlag("render.since", renderCmd.Flags().Lookup("since"))
	_ = viper.BindPFlag("render.until", renderCmd.Flags().Lookup("until"))
	_ = viper.BindPFlag("render.limit", renderCmd.Flags().Lookup("limit"))
	_ = viper.BindPFlag("render.include", renderCmd.Flags().Lookup("include"))
	_ = viper.BindPFlag("render.output", renderCmd.Flags().Lookup("output"))
	_ = viper.BindPFlag("render.template", renderCmd.Flags().Lookup("template"))
}

func runRender(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	since, err := parseSinceFlag(viper.GetString("render.since"), time.Now())
	if err != nil {
		return err
	}
	until, err := parseUntilFlag(viper.GetString("render.until"))
	if err != nil {
		return err
	}
	limit := viper.GetInt("render.limit")

	includeHistory, includeStarred, err := parseIncludeFlag(viper.GetString("render.include"))
	if err != nil {
		return err
	}

	db, err := database.New(cfg.Database)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	var history, starred []render.Episode

	if includeHistory {
		eps, err := db.HistoryEpisodes(since, until, limit)
		if err != nil {
			return fmt.Errorf("query history: %w", err)
		}
		history = toRenderEpisodes(eps)
	}
	if includeStarred {
		eps, err := db.StarredEpisodes(limit)
		if err != nil {
			return fmt.Errorf("query starred: %w", err)
		}
		starred = toRenderEpisodes(eps)
	}

	md, err := render.Render(history, starred, render.Options{
		TemplatePath: viper.GetString("render.template"),
	})
	if err != nil {
		return err
	}

	out := viper.GetString("render.output")
	if out == "" {
		fmt.Print(md)
		return nil
	}
	if err := os.WriteFile(out, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	return nil
}

// parseSinceFlag accepts either a Go duration ("168h") subtracted from
// `now`, or a YYYY-MM-DD date treated as midnight local time. An empty
// value returns nil (no lower bound).
func parseSinceFlag(s string, now time.Time) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		t := now.Add(-d)
		return &t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return &t, nil
	}
	return nil, fmt.Errorf("invalid --since %q: expected Go duration (e.g. 168h) or date (YYYY-MM-DD)", s)
}

// parseUntilFlag accepts a YYYY-MM-DD date treated as end-of-day local
// time, or empty for no upper bound.
func parseUntilFlag(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil, fmt.Errorf("invalid --until %q: expected date (YYYY-MM-DD)", s)
	}
	// End-of-day so the bound is inclusive.
	t = t.Add(24*time.Hour - time.Second)
	return &t, nil
}

// parseIncludeFlag returns (history, starred) booleans from a
// comma-separated list. Unknown tokens are rejected.
func parseIncludeFlag(s string) (bool, bool, error) {
	if s == "" {
		return false, false, errors.New("--include must list at least one section")
	}
	var history, starred bool
	for _, tok := range strings.Split(s, ",") {
		switch strings.TrimSpace(tok) {
		case "history":
			history = true
		case "starred":
			starred = true
		case "":
			// skip empty tokens from trailing commas
		default:
			return false, false, fmt.Errorf("unknown --include token %q (valid: history, starred)", tok)
		}
	}
	if !history && !starred {
		return false, false, errors.New("--include must list at least one section")
	}
	return history, starred, nil
}

func toRenderEpisodes(eps []database.Episode) []render.Episode {
	out := make([]render.Episode, len(eps))
	for i, e := range eps {
		out[i] = render.FromAPI(
			e.UUID, e.Title, e.URL,
			e.PodcastUUID, e.PodcastTitle, e.Published,
			e.PlayedUpTo, e.Duration,
		)
	}
	return out
}
