package cmd

import (
	"fmt"
	"os"

	"github.com/nexspence/nxs/internal/client"
	"github.com/nexspence/nxs/internal/config"
	"github.com/nexspence/nxs/internal/output"
	"github.com/spf13/cobra"
)

var (
	flagJSON    bool
	flagPlain   bool
	flagURL     string
	flagToken   string
	flagContext string

	version = "dev" // overridden by goreleaser ldflags

	cfg       *config.Config
	nxsClient *client.Client
	printer   output.Printer
)

var rootCmd = &cobra.Command{
	Use:     "nxs",
	Short:   "CLI for Nexspence artifact repository manager",
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(flagContext)
		if err != nil {
			return err
		}
		url := cfg.CurrentURL()
		token := cfg.CurrentToken()
		if flagURL != "" {
			url = flagURL
		}
		if flagToken != "" {
			token = flagToken
		}
		printer = output.NewPrinter(flagJSON, flagPlain)
		if url != "" {
			nxsClient = client.New(url, token)
		}
		return nil
	},
}

// requireClient returns an error if the server URL is not configured.
// Call this at the start of any RunE that makes API calls.
func requireClient() error {
	if nxsClient == nil {
		return fmt.Errorf("not logged in — run 'nxs login' or set NXS_URL")
	}
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&flagPlain, "plain", false, "Plain text output (no colors)")
	rootCmd.PersistentFlags().StringVar(&flagURL, "url", "", "Override server URL")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "Override auth token")
	rootCmd.PersistentFlags().StringVar(&flagContext, "context", "", "Use named context")
}
