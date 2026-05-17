package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage role assignments",
}

var roleAssignCmd = &cobra.Command{
	Use:   "assign <username> <role>",
	Short: "Assign a role to a user",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		if err := nxsClient.RoleAssign(args[0], []string{args[1]}); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Role %q assigned to user %q", args[1], args[0]))
		return nil
	},
}

func init() {
	roleCmd.AddCommand(roleAssignCmd)
	rootCmd.AddCommand(roleCmd)
}
