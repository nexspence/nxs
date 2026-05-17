package cmd

import (
	"fmt"
	"syscall"

	"github.com/nexspence/nxs/internal/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate from a Nexus-compatible source",
}

var migrateFromCmd = &cobra.Command{
	Use:   "from <source-url>",
	Short: "Start a migration job from a Nexus source",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		user, _ := cmd.Flags().GetString("user")
		repos, _ := cmd.Flags().GetBool("repos")
		users, _ := cmd.Flags().GetBool("users")
		blobs, _ := cmd.Flags().GetBool("blobs")

		fmt.Print("Source password: ")
		raw, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return err
		}

		job, err := nxsClient.MigrateStart(client.MigrateRequest{
			SourceURL: args[0],
			Username:  user,
			Password:  string(raw),
			Repos:     repos,
			Users:     users,
			Blobs:     blobs,
		})
		if err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Migration job started (id: %s)", job.ID))
		return nil
	},
}

func init() {
	migrateFromCmd.Flags().String("user", "admin", "Source server username")
	migrateFromCmd.Flags().Bool("repos", true, "Migrate repository definitions")
	migrateFromCmd.Flags().Bool("users", false, "Migrate users")
	migrateFromCmd.Flags().Bool("blobs", false, "Migrate artifacts (blobs)")

	migrateCmd.AddCommand(migrateFromCmd)
	rootCmd.AddCommand(migrateCmd)
}
