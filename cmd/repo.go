package cmd

import (
	"fmt"
	"strings"

	"github.com/nexspence/nxs/internal/client"
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

// repoUpdateOptions turns the flags of "repo update" into the options the
// client sends. A flag that was not given stays unset, so the setting it
// would change is left as it is.
func repoUpdateOptions(cmd *cobra.Command) (client.RepoUpdateOptions, error) {
	flags := cmd.Flags()
	var opts client.RepoUpdateOptions
	given := 0

	online, _ := flags.GetBool("online")
	offline, _ := flags.GetBool("offline")
	if online && offline {
		return opts, fmt.Errorf("--online and --offline contradict each other")
	}
	if flags.Changed("online") || flags.Changed("offline") {
		v := online
		opts.Online = &v
		given++
	}

	if flags.Changed("description") {
		v, _ := flags.GetString("description")
		opts.Description = &v
		given++
	}
	if flags.Changed("write-policy") {
		v, _ := flags.GetString("write-policy")
		switch v {
		case "allow", "allow_once", "deny":
		default:
			return opts, fmt.Errorf("--write-policy must be one of allow, allow_once, deny")
		}
		opts.WritePolicy = &v
		given++
	}
	if flags.Changed("allow-redeploy-latest") {
		v, _ := flags.GetBool("allow-redeploy-latest")
		opts.AllowRedeployLatest = &v
		given++
	}
	if flags.Changed("pull-block-severity") {
		v, _ := flags.GetString("pull-block-severity")
		switch v {
		case "off":
			v = ""
		case "low", "medium", "high", "critical", "malicious":
		default:
			return opts, fmt.Errorf("--pull-block-severity must be one of low, medium, high, critical, malicious, off")
		}
		opts.PullBlockSeverity = &v
		given++
	}
	members, _ := flags.GetStringArray("member")
	if len(members) > 0 {
		opts.MemberNames = members
		given++
	}

	if given == 0 {
		return opts, fmt.Errorf("no update given — pass at least one of --online, --offline, --description, --write-policy, --allow-redeploy-latest, --pull-block-severity, --member")
	}
	return opts, nil
}

var repoUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		opts, err := repoUpdateOptions(cmd)
		if err != nil {
			return err
		}
		if err := nxsClient.RepoUpdate(args[0], opts); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Repository %q updated", args[0]))
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

	repoUpdateCmd.Flags().Bool("online", false, "Bring the repository online")
	repoUpdateCmd.Flags().Bool("offline", false, "Take the repository offline")
	repoUpdateCmd.Flags().String("description", "", "Description of the repository")
	repoUpdateCmd.Flags().String("write-policy", "", "Write policy: allow, allow_once, deny")
	repoUpdateCmd.Flags().Bool("allow-redeploy-latest", false, "Allow the latest tag to be re-pushed (use --allow-redeploy-latest=false to refuse it)")
	repoUpdateCmd.Flags().String("pull-block-severity", "", "Severity at which a pull is blocked: low, medium, high, critical, malicious, off")
	repoUpdateCmd.Flags().StringArray("member", nil, "Member repository of a group (repeatable)")

	repoCmd.AddCommand(repoListCmd, repoCreateCmd, repoDeleteCmd, repoInfoCmd, repoUpdateCmd)
	rootCmd.AddCommand(repoCmd)
}
