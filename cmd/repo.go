package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories",
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		format, _ := cmd.Flags().GetString("format")
		repoType, _ := cmd.Flags().GetString("type")
		repos, err := nxsClient.RepoList(format, repoType)
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(repos)
			return nil
		}
		rows := make([][]string, 0, len(repos))
		for _, r := range repos {
			status := "● online"
			if !r.Online {
				status = "○ offline"
			}
			rows = append(rows, []string{r.Name, r.Format, r.Type, status})
		}
		printer.Table([]string{"NAME", "FORMAT", "TYPE", "STATUS"}, rows)
		return nil
	},
}

var repoCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		format, _ := cmd.Flags().GetString("format")
		repoType, _ := cmd.Flags().GetString("type")
		blobStore, _ := cmd.Flags().GetString("blob-store")
		if format == "" || repoType == "" {
			return fmt.Errorf("--format and --type are required")
		}
		if err := nxsClient.RepoCreate(args[0], format, repoType, blobStore); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Repository %q created", args[0]))
		return nil
	},
}

var repoDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Delete repository %q? This cannot be undone. [y/N]: ", args[0])
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}
		if err := nxsClient.RepoDelete(args[0]); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Repository %q deleted", args[0]))
		return nil
	},
}

var repoInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show repository details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		repo, err := nxsClient.RepoInfo(args[0])
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(repo)
			return nil
		}
		printer.Table([]string{"FIELD", "VALUE"}, [][]string{
			{"Name", repo.Name},
			{"Format", repo.Format},
			{"Type", repo.Type},
			{"URL", repo.URL},
			{"Online", fmt.Sprintf("%v", repo.Online)},
		})
		return nil
	},
}

func init() {
	repoListCmd.Flags().String("format", "", "Filter by format (maven2, npm, docker, ...)")
	repoListCmd.Flags().String("type", "", "Filter by type (hosted, proxy, group)")

	repoCreateCmd.Flags().String("format", "", "Repository format (required)")
	repoCreateCmd.Flags().String("type", "", "Repository type: hosted, proxy, group (required)")
	repoCreateCmd.Flags().String("blob-store", "default", "Blob store name")

	repoDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	repoCmd.AddCommand(repoListCmd, repoCreateCmd, repoDeleteCmd, repoInfoCmd)
	rootCmd.AddCommand(repoCmd)
}
