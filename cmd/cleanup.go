package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Manage cleanup policies",
}

var cleanupRunCmd = &cobra.Command{
	Use:   "run <policy-name>",
	Short: "Execute a cleanup policy immediately",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		if err := nxsClient.CleanupRun(args[0]); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Cleanup policy %q triggered", args[0]))
		return nil
	},
}

func init() {
	cleanupCmd.AddCommand(cleanupRunCmd)
	rootCmd.AddCommand(cleanupCmd)
}
