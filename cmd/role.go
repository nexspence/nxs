package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage role assignments",
}

var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the roles defined on the server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		roles, err := nxsClient.RoleList()
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(roles)
			return nil
		}
		rows := make([][]string, 0, len(roles))
		for _, r := range roles {
			rows = append(rows, []string{r.Name, r.Description})
		}
		printer.Table([]string{"ROLE", "DESCRIPTION"}, rows)
		return nil
	},
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

var roleRevokeCmd = &cobra.Command{
	Use:   "revoke <username> <role>",
	Short: "Revoke a role from a user",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		if err := nxsClient.RoleRevoke(args[0], []string{args[1]}); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Role %q revoked from user %q", args[1], args[0]))
		return nil
	},
}

func init() {
	roleCmd.AddCommand(roleListCmd, roleAssignCmd, roleRevokeCmd)
	rootCmd.AddCommand(roleCmd)
}
