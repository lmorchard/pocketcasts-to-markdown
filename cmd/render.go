package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// renderCmd reads episodes from the local archive and emits Markdown.
var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render Markdown from the local archive",
	Long: `Read episodes from the local SQLite archive and emit a Markdown
document summarizing recent activity. No network calls.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger()
		log.Warnf("render: not yet implemented (since=%s, output=%s)",
			viper.GetString("render.since"), viper.GetString("render.output"))
		return nil
	},
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
