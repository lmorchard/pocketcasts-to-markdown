package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmorchard/pocketcasts-to-markdown/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	log     = logrus.New()
	cfg     *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pocketcasts-to-markdown",
	Short: "Fetch Pocket Casts activity and emit Markdown",
	Long: `pocketcasts-to-markdown syncs your Pocket Casts listening history
and starred episodes into a local SQLite archive, then renders Markdown
summaries from that archive over arbitrary date ranges.

Designed for unattended use (cron / scheduler). Credentials are read from
environment variables (POCKETCASTS_EMAIL, POCKETCASTS_PASSWORD) or a YAML
config file; the auth token is cached locally and refreshed on demand.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initConfig()
		setupLogging()
	},
}

// Execute adds all child commands to the root command and sets appropriate flags.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Configuration file flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $XDG_CONFIG_HOME/pocketcasts-to-markdown/pocketcasts-to-markdown.yaml)")

	// Logging flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().Bool("debug", false, "debug output")
	rootCmd.PersistentFlags().Bool("log-json", false, "output logs in JSON format")

	// Database flag
	rootCmd.PersistentFlags().String("database", "", "database file path (default: $XDG_STATE_HOME/pocketcasts-to-markdown/state.db)")

	// Bind flags to viper
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("log_json", rootCmd.PersistentFlags().Lookup("log-json"))
	_ = viper.BindPFlag("database", rootCmd.PersistentFlags().Lookup("database"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Search XDG_CONFIG_HOME and CWD for pocketcasts-to-markdown.yaml
		viper.AddConfigPath(xdgConfigDir())
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("pocketcasts-to-markdown")
	}

	// Defaults
	viper.SetDefault("database", defaultDatabasePath())
	viper.SetDefault("verbose", false)
	viper.SetDefault("debug", false)
	viper.SetDefault("log_json", false)

	// Environment variables: POCKETCASTS_EMAIL, POCKETCASTS_PASSWORD, etc.
	// Replace dots with underscores so nested keys map to env vars.
	viper.SetEnvPrefix("POCKETCASTS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Explicit bindings for credentials so they read POCKETCASTS_EMAIL / POCKETCASTS_PASSWORD
	_ = viper.BindEnv("email", "POCKETCASTS_EMAIL")
	_ = viper.BindEnv("password", "POCKETCASTS_PASSWORD")

	// Read config file if available
	if err := viper.ReadInConfig(); err != nil {
		if cfgFile != "" {
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
			os.Exit(1)
		}
	}
}

// setupLogging configures the logger based on configuration
func setupLogging() {
	if viper.GetBool("log_json") {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	if viper.GetBool("debug") {
		log.SetLevel(logrus.DebugLevel)
	} else if viper.GetBool("verbose") {
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetLevel(logrus.WarnLevel)
	}
}

// GetConfig returns the application configuration, loading it if necessary
func GetConfig() *config.Config {
	if cfg == nil {
		cfg = &config.Config{
			Database: viper.GetString("database"),
			Verbose:  viper.GetBool("verbose"),
			Debug:    viper.GetBool("debug"),
			LogJSON:  viper.GetBool("log_json"),
			Email:    viper.GetString("email"),
			Password: viper.GetString("password"),
		}
	}
	return cfg
}

// GetLogger returns the configured logger
func GetLogger() *logrus.Logger {
	return log
}

// xdgConfigDir returns $XDG_CONFIG_HOME/pocketcasts-to-markdown,
// falling back to ~/.config/pocketcasts-to-markdown.
func xdgConfigDir() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "pocketcasts-to-markdown")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config", "pocketcasts-to-markdown")
}

// xdgStateDir returns $XDG_STATE_HOME/pocketcasts-to-markdown,
// falling back to ~/.local/state/pocketcasts-to-markdown.
func xdgStateDir() string {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return filepath.Join(v, "pocketcasts-to-markdown")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".local", "state", "pocketcasts-to-markdown")
}

func defaultDatabasePath() string {
	return filepath.Join(xdgStateDir(), "state.db")
}
