package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var componentCmd = &cobra.Command{
	Use:   "component",
	Short: "Manage components",
}

var componentDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a component",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Delete component %q? This cannot be undone. [y/N]: ", args[0])
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}
		if err := nxsClient.ComponentDelete(args[0]); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Component %q deleted", args[0]))
		return nil
	},
}

func init() {
	componentDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	componentCmd.AddCommand(componentDeleteCmd)
	rootCmd.AddCommand(componentCmd)
}
