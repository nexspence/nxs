package cmd

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/nexspence/nxs/internal/client"
	"github.com/nexspence/nxs/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate to a Nexspence server",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		user, _ := cmd.Flags().GetString("user")
		contextName, _ := cmd.Flags().GetString("context")

		if url == "" {
			fmt.Print("Server URL: ")
			fmt.Scanln(&url)
		}
		url = strings.TrimRight(url, "/")
		if user == "" {
			fmt.Print("Username: ")
			fmt.Scanln(&user)
		}
		fmt.Print("Password: ")
		raw, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}

		token, err := client.Login(url, user, string(raw))
		if err != nil {
			return err
		}

		cfgPath := os.Getenv("NXS_CONFIG")
		existing, _ := config.Load("")
		if existing == nil {
			existing = &config.Config{}
		}
		if err := existing.Save(cfgPath, contextName, url, token); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Logged in to %s as %s (context: %s)\n", url, user, contextName)
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved credentials for the active context",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(flagContext)
		if err != nil {
			return err
		}
		cfgPath := os.Getenv("NXS_CONFIG")
		if err := cfg.Save(cfgPath, cfg.ActiveContext(), cfg.CurrentURL(), ""); err != nil {
			return err
		}
		fmt.Printf("Logged out from context %q\n", cfg.ActiveContext())
		return nil
	},
}

func init() {
	loginCmd.Flags().String("url", "", "Server URL")
	loginCmd.Flags().String("user", "", "Username")
	loginCmd.Flags().String("context", "default", "Context name to save credentials under")
	rootCmd.AddCommand(loginCmd, logoutCmd)
}
