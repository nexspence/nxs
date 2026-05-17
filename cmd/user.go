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

func init() {
	userCreateCmd.Flags().String("email", "", "Email address (required)")
	userCreateCmd.Flags().String("first-name", "", "First name")
	userCreateCmd.Flags().String("last-name", "", "Last name")
	userCreateCmd.Flags().String("password", "", "Initial password (required)")
	userCreateCmd.Flags().StringSlice("role", nil, "Role to assign (repeatable)")

	userCmd.AddCommand(userListCmd, userCreateCmd)
	rootCmd.AddCommand(userCmd)
}
