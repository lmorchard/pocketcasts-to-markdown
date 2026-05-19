package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// exportCmd is the orchestrator-facing entry point with the canonical
// `--since/--until/-o` flag shape shared across all *-to-markdown tools.
// It composes `sync` followed by `render` over the given window.
//
// Pocket Casts' existing `render` subcommand already accepts the canonical
// flag shape (Go duration or YYYY-MM-DD for --since, YYYY-MM-DD for
// --until), so export simply overrides the viper keys render reads and
// delegates.
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Orchestrator-friendly sync + render in one shot",
	Long: `Pull recent Pocket Casts activity into the local archive and render
markdown over the given time window in one invocation.

The --since/--until flag shape matches the contract used by me-to-markdown
and the rest of the *-to-markdown tools. Section selection, item limits,
and template options are read from the config file or environment; this
subcommand exposes only the orchestrator-facing flags.

Example usage:
  pocketcasts-to-markdown export --since 168h
  pocketcasts-to-markdown export --since 2026-05-11 --until 2026-05-18 -o pc.md`,
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")
		output, _ := cmd.Flags().GetString("output")

		// Override the viper keys the render subcommand reads. Render's
		// parseSinceFlag already accepts the canonical Go-duration or
		// YYYY-MM-DD shapes, and parseUntilFlag accepts YYYY-MM-DD.
		viper.Set("render.since", since)
		viper.Set("render.until", until)
		viper.Set("render.output", output)

		if err := syncCmd.RunE(cmd, args); err != nil {
			return err
		}
		return renderCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().String("since", "", "Start of time window (YYYY-MM-DD or Go duration like 168h) — required")
	exportCmd.Flags().String("until", "", "End of time window (YYYY-MM-DD, defaults to no upper bound)")
	exportCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	_ = exportCmd.MarkFlagRequired("since")
}
