package cmd

import (
	"fmt"
	"strings"

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

var roleCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a role",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		if err := nxsClient.RoleCreate(args[0]); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Role %q created", args[0]))
		return nil
	},
}

var roleUpdateCmd = &cobra.Command{
	Use:   "update <name> --name NEW-NAME",
	Short: "Rename a role",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		newName, _ := cmd.Flags().GetString("name")
		if newName == "" {
			return fmt.Errorf("--name is required")
		}
		if err := nxsClient.RoleUpdate(args[0], newName); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Role %q renamed to %q", args[0], newName))
		return nil
	},
}

var roleDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a role",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Delete role %q? This cannot be undone. [y/N]: ", args[0])
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}
		if err := nxsClient.RoleDelete(args[0]); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Role %q deleted", args[0]))
		return nil
	},
}

func init() {
	roleUpdateCmd.Flags().String("name", "", "New name for the role (required)")
	roleDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	roleCmd.AddCommand(roleListCmd, roleCreateCmd, roleUpdateCmd, roleDeleteCmd, roleAssignCmd, roleRevokeCmd)
	rootCmd.AddCommand(roleCmd)
}
