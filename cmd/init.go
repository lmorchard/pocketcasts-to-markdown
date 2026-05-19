package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lmorchard/pocketcasts-to-markdown/internal/templates"
	"github.com/spf13/cobra"
)

const defaultConfigContent = `# Configuration file for pocketcasts-to-markdown
# Prefer environment variables (POCKETCASTS_EMAIL, POCKETCASTS_PASSWORD)
# over committing credentials to a file.

email: ""
password: ""

# Path to the local SQLite archive.
# Default: $XDG_STATE_HOME/pocketcasts-to-markdown/state.db
# database: "/path/to/state.db"

verbose: false
debug: false
log_json: false
`

// initCmd creates a default config file and a customizable markdown template.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a config file and a customizable markdown template",
	Long: `Create a starter config file and a customizable markdown template
in the current directory.

Use --force to overwrite existing files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger()
		force, _ := cmd.Flags().GetBool("force")
		templateFile, _ := cmd.Flags().GetString("template-file")
		configFile := "pocketcasts-to-markdown.yaml"

		if err := writeFile(configFile, []byte(defaultConfigContent), force); err != nil {
			return err
		}
		log.Infof("Wrote %s", configFile)

		tpl, err := templates.GetDefaultTemplate()
		if err != nil {
			return fmt.Errorf("failed to load default template: %w", err)
		}
		if err := writeFile(templateFile, []byte(tpl), force); err != nil {
			return err
		}
		log.Infof("Wrote %s", templateFile)

		fmt.Println()
		fmt.Println("Initialization complete. Next steps:")
		fmt.Printf("  1. Edit %s (or set POCKETCASTS_EMAIL / POCKETCASTS_PASSWORD)\n", configFile)
		fmt.Printf("  2. (Optional) Customize %s\n", templateFile)
		fmt.Println("  3. Run: pocketcasts-to-markdown sync")
		fmt.Println("  4. Run: pocketcasts-to-markdown render --since 168h")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().Bool("force", false, "Overwrite existing files")
	initCmd.Flags().String("template-file", "pocketcasts-to-markdown.md", "Markdown template file to create")
}

func writeFile(path string, content []byte, force bool) error {
	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", filepath.Clean(path))
	}
	return os.WriteFile(path, content, 0o644)
}
