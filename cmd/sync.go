package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lmorchard/pocketcasts-to-markdown/internal/database"
	"github.com/lmorchard/pocketcasts-to-markdown/internal/pocketcasts"
	"github.com/spf13/cobra"
)

const (
	kvAuthToken    = "auth.token"
	kvSyncLastRun  = "sync.last_run_at"
	syncCmdTimeout = 2 * time.Minute
)

// syncCmd fetches recent activity from Pocket Casts and upserts it
// into the local SQLite archive.
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch recent activity from Pocket Casts into the local archive",
	Long: `Log in to Pocket Casts (using a cached token when available),
fetch the listening history and starred episodes, and upsert them into
the local SQLite archive at --database.

This command is idempotent: running it repeatedly is the intended way
to keep the archive current. The archive grows over time; episodes are
never removed when they fall out of the API's recent window.`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	cfg := GetConfig()

	if cfg.Email == "" || cfg.Password == "" {
		return errors.New("POCKETCASTS_EMAIL and POCKETCASTS_PASSWORD must be set (env or config file)")
	}

	db, err := database.New(cfg.Database)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	client := pocketcasts.New()

	// Restore cached token, if any. A 401 below will trigger a relogin.
	cachedToken, hasCached, err := db.GetKV(kvAuthToken)
	if err != nil {
		return fmt.Errorf("read cached token: %w", err)
	}
	if hasCached {
		log.Debug("using cached auth token")
		client.SetToken(cachedToken)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), syncCmdTimeout)
	defer cancel()

	saveToken := func(t string) error { return db.SetKV(kvAuthToken, t) }

	historyN, err := syncHistory(ctx, log, db, client, cfg.Email, cfg.Password, saveToken)
	if err != nil {
		return err
	}

	starredN, err := syncStarred(ctx, log, db, client, cfg.Email, cfg.Password, saveToken)
	if err != nil {
		return err
	}

	if err := db.SetKV(kvSyncLastRun, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("record sync timestamp: %w", err)
	}

	fmt.Printf("sync: %d history episode(s), %d starred episode(s)\n", historyN, starredN)
	return nil
}

// syncHistory fetches history, retrying once after a fresh login on 401.
// Returns the number of episodes upserted.
func syncHistory(
	ctx context.Context,
	log loggerLike,
	db *database.DB,
	client *pocketcasts.Client,
	email, password string,
	saveToken func(string) error,
) (int, error) {
	episodes, err := fetchWithRelogin(ctx, log, client, email, password, saveToken, client.History)
	if err != nil {
		return 0, fmt.Errorf("fetch history: %w", err)
	}
	for _, ep := range episodes {
		if err := db.UpsertEpisode(toDBEpisode(ep), database.SourceHistory); err != nil {
			return 0, err
		}
	}
	log.Infof("upserted %d history episode(s)", len(episodes))
	return len(episodes), nil
}

// syncStarred fetches starred episodes. A 404 is logged but not fatal —
// the endpoint path is a guess (see spec.md open questions); a real run
// will tell us whether /user/starred is right.
func syncStarred(
	ctx context.Context,
	log loggerLike,
	db *database.DB,
	client *pocketcasts.Client,
	email, password string,
	saveToken func(string) error,
) (int, error) {
	episodes, err := fetchWithRelogin(ctx, log, client, email, password, saveToken, client.Starred)
	if err != nil {
		var apiErr *pocketcasts.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			log.Warnf("starred endpoint /user/starred returned 404 — endpoint path may differ; skipping")
			return 0, nil
		}
		return 0, fmt.Errorf("fetch starred: %w", err)
	}
	for _, ep := range episodes {
		if err := db.UpsertEpisode(toDBEpisode(ep), database.SourceStarred); err != nil {
			return 0, err
		}
	}
	log.Infof("upserted %d starred episode(s)", len(episodes))
	return len(episodes), nil
}

// fetchWithRelogin runs fn(), and if it fails with 401, logs in fresh
// (persisting the new token via saveToken) and retries once.
func fetchWithRelogin(
	ctx context.Context,
	log loggerLike,
	client *pocketcasts.Client,
	email, password string,
	saveToken func(string) error,
	fn func(context.Context) ([]pocketcasts.Episode, error),
) ([]pocketcasts.Episode, error) {
	if client.Token() != "" {
		eps, err := fn(ctx)
		if err == nil {
			return eps, nil
		}
		if !pocketcasts.IsUnauthorized(err) {
			return nil, err
		}
		log.Debug("cached token rejected (401); logging in fresh")
	}

	token, err := client.Login(ctx, email, password)
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	if err := saveToken(token); err != nil {
		return nil, fmt.Errorf("cache token: %w", err)
	}
	return fn(ctx)
}

// toDBEpisode converts the API DTO to the storage DTO.
func toDBEpisode(ep pocketcasts.Episode) database.Episode {
	return database.Episode{
		UUID:         ep.UUID,
		PodcastUUID:  ep.PodcastUUID,
		PodcastTitle: ep.PodcastTitle,
		Title:        ep.Title,
		URL:          ep.URL,
		Published:    ep.Published,
		PlayedUpTo:   ep.PlayedUpTo,
		Duration:     ep.Duration,
	}
}

// loggerLike captures the subset of logrus.Logger we use. Keeps the
// internal helpers in this file independent of the concrete logger.
type loggerLike interface {
	Debug(args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
}
