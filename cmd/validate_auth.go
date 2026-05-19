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

const validateAuthTimeout = 30 * time.Second

// validateAuthCmd answers "do my credentials work?" with a single-line
// stdout and a clean exit code. Suitable for orchestrators or scripts
// gating behavior on auth health.
var validateAuthCmd = &cobra.Command{
	Use:   "validate-auth",
	Short: "Check whether the cached Pocket Casts bearer token is accepted",
	Long: `Hit a minimal authenticated endpoint with the cached bearer token (or
freshly log in with POCKETCASTS_EMAIL / POCKETCASTS_PASSWORD) and
exit 0 if the token is accepted, non-zero otherwise.

If no cached token exists and no credentials are configured, fails
with a hint to run ` + "`login`" + `.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := GetConfig()

		db, err := database.New(cfg.Database)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer func() { _ = db.Close() }()

		client := pocketcasts.New()

		cachedToken, hasCached, err := db.GetKV(kvAuthToken)
		if err != nil {
			return fmt.Errorf("read cached token: %w", err)
		}
		if hasCached {
			client.SetToken(cachedToken)
		} else if cfg.Email == "" || cfg.Password == "" {
			return errors.New("no cached auth token and no credentials configured; run `pocketcasts-to-markdown login` (or set POCKETCASTS_EMAIL / POCKETCASTS_PASSWORD)")
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), validateAuthTimeout)
		defer cancel()

		// Use fetchWithRelogin so an expired cached token gets refreshed
		// in-place (matching what sync/export do). We discard the
		// resulting episodes — we only care about whether auth worked.
		_, err = fetchWithRelogin(ctx, GetLogger(), client, cfg.Email, cfg.Password,
			func(t string) error { return db.SetKV(kvAuthToken, t) },
			client.History,
		)
		if err != nil {
			return fmt.Errorf("pocketcasts auth check: %w", err)
		}

		identity := cfg.Email
		if identity == "" {
			identity = "cached token"
		}
		fmt.Printf("validate-auth: ok (%s)\n", identity)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateAuthCmd)
}
