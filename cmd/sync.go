package cmd

import (
	"github.com/spf13/cobra"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger()
		log.Warn("sync: not yet implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
