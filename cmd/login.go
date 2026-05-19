package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/lmorchard/pocketcasts-to-markdown/internal/database"
	"github.com/lmorchard/pocketcasts-to-markdown/internal/pocketcasts"
	"github.com/spf13/cobra"
)

const loginCmdTimeout = 30 * time.Second

// loginCmd authenticates with Pocket Casts and caches the resulting
// token in the local archive's kv table. After login, `sync` can run
// without any credentials in env or config — until the token expires.
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Pocket Casts and cache an auth token",
	Long: `Authenticate with Pocket Casts and store the resulting bearer
token in the local SQLite archive so subsequent runs of sync don't need
credentials.

Credentials, in order of precedence:
  1. --email / --password flags
  2. POCKETCASTS_EMAIL / POCKETCASTS_PASSWORD env vars
  3. email / password in the config file
  4. --password-stdin reads the password from STDIN (recommended for
     scripts: avoids exposing it in shell history)

If the cached token expires, this command must be re-run.`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().String("email", "", "Pocket Casts account email")
	loginCmd.Flags().String("password", "", "Pocket Casts account password (insecure: visible in shell history)")
	loginCmd.Flags().Bool("password-stdin", false, "Read password from STDIN (one line)")
}

func runLogin(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	cfg := GetConfig()

	email, _ := cmd.Flags().GetString("email")
	if email == "" {
		email = cfg.Email
	}
	if email == "" {
		return errors.New("email is required: pass --email, set POCKETCASTS_EMAIL, or put it in the config file")
	}

	password, _ := cmd.Flags().GetString("password")
	if password == "" {
		password = cfg.Password
	}

	passwordStdin, _ := cmd.Flags().GetBool("password-stdin")
	if passwordStdin {
		if password != "" {
			return errors.New("--password-stdin conflicts with --password / config password")
		}
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read password from stdin: %w", err)
		}
		password = strings.TrimRight(string(b), "\r\n")
	}

	if password == "" {
		return errors.New("password is required: pass --password, --password-stdin, set POCKETCASTS_PASSWORD, or put it in the config file")
	}

	db, err := database.New(cfg.Database)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithTimeout(cmd.Context(), loginCmdTimeout)
	defer cancel()

	client := pocketcasts.New()
	token, err := client.Login(ctx, email, password)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	if err := db.SetKV(kvAuthToken, token); err != nil {
		return fmt.Errorf("cache token: %w", err)
	}

	log.Infof("logged in as %s; token cached", email)
	fmt.Println("login successful; token cached")
	return nil
}
