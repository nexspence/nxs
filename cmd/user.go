package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		users, err := nxsClient.UserList()
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(users)
			return nil
		}
		rows := make([][]string, 0, len(users))
		for _, u := range users {
			rows = append(rows, []string{
				u.UserID,
				strings.TrimSpace(u.FirstName + " " + u.LastName),
				u.EmailAddress,
				strings.Join(u.Roles, ","),
			})
		}
		printer.Table([]string{"USER", "NAME", "EMAIL", "ROLES"}, rows)
		return nil
	},
}

var userCreateCmd = &cobra.Command{
	Use:   "create <username>",
	Short: "Create a new user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		email, _ := cmd.Flags().GetString("email")
		firstName, _ := cmd.Flags().GetString("first-name")
		lastName, _ := cmd.Flags().GetString("last-name")
		password, _ := cmd.Flags().GetString("password")
		roles, _ := cmd.Flags().GetStringSlice("role")
		if email == "" {
			return fmt.Errorf("--email is required")
		}
		if password == "" {
			return fmt.Errorf("--password is required")
		}
		if err := nxsClient.UserCreate(args[0], firstName, lastName, email, password, roles); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("User %q created", args[0]))
		return nil
	},
}

var userUpdateCmd = &cobra.Command{
	Use:   "update <username>",
	Short: "Update a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		firstName, _ := cmd.Flags().GetString("first-name")
		lastName, _ := cmd.Flags().GetString("last-name")
		email, _ := cmd.Flags().GetString("email")
		roles, _ := cmd.Flags().GetStringSlice("role")
		if firstName == "" && lastName == "" && email == "" && len(roles) == 0 {
			return fmt.Errorf("nothing to update — give --first-name, --last-name, --email or --role")
		}
		if err := nxsClient.UserUpdate(args[0], firstName, lastName, email, roles); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("User %q updated", args[0]))
		return nil
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <username>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Delete user %q? This cannot be undone. [y/N]: ", args[0])
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}
		if err := nxsClient.UserDelete(args[0]); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("User %q deleted", args[0]))
		return nil
	},
}

var userPasswordCmd = &cobra.Command{
	Use:   "password <username>",
	Short: "Change a user's password",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		password, _ := cmd.Flags().GetString("password")
		if password == "" {
			return fmt.Errorf("--password is required")
		}
		if err := nxsClient.UserChangePassword(args[0], password); err != nil {
			return err
		}
		// The password is a secret: it is never echoed, not even here.
		printer.Success(fmt.Sprintf("Password changed for user %q", args[0]))
		return nil
	},
}

func init() {
	userCreateCmd.Flags().String("email", "", "Email address (required)")
	userCreateCmd.Flags().String("first-name", "", "First name")
	userCreateCmd.Flags().String("last-name", "", "Last name")
	userCreateCmd.Flags().String("password", "", "Initial password (required)")
	userCreateCmd.Flags().StringSlice("role", nil, "Role to assign (repeatable)")

	userUpdateCmd.Flags().String("first-name", "", "New first name")
	userUpdateCmd.Flags().String("last-name", "", "New last name")
	userUpdateCmd.Flags().String("email", "", "New email address")
	userUpdateCmd.Flags().StringSlice("role", nil, "Role to add to the user (repeatable)")

	userDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	userPasswordCmd.Flags().String("password", "", "New password (required)")

	userCmd.AddCommand(userListCmd, userCreateCmd, userUpdateCmd, userDeleteCmd, userPasswordCmd)
	rootCmd.AddCommand(userCmd)
}
