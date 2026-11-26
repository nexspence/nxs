package cmd

import (
	"fmt"
	"strings"

	"github.com/nexspence/nxs/internal/client"
	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Manage cleanup policies",
}

var cleanupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cleanup policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		policies, err := nxsClient.CleanupList()
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(policies)
			return nil
		}
		rows := make([][]string, 0, len(policies))
		for _, p := range policies {
			rows = append(rows, []string{
				p.Name,
				p.ID,
				p.Format,
				cleanupCriteriaSummary(p),
				p.ScheduleCron,
				p.Scope.RepositoryName,
				fmt.Sprintf("%v", p.DryRun),
				fmt.Sprintf("%v", p.Enabled),
			})
		}
		printer.Table([]string{"NAME", "ID", "FORMAT", "CRITERIA", "CRON", "REPOSITORY", "DRY RUN", "ENABLED"}, rows)
		return nil
	},
}

// cleanupCriteriaSummary describes a policy's criteria in one column, the way
// the server reads them: an age in days, a download age in days, a path prefix
// and a name glob.
func cleanupCriteriaSummary(p client.CleanupPolicy) string {
	parts := make([]string, 0, 4)
	if days, ok := cleanupIntCriteria(p.Criteria, "artifactAgeDays"); ok {
		parts = append(parts, fmt.Sprintf("age>%dd", days))
	}
	if days, ok := cleanupIntCriteria(p.Criteria, "lastDownloadedDays"); ok {
		parts = append(parts, fmt.Sprintf("unused>%dd", days))
	}
	if prefix, ok := p.Criteria["pathPrefix"].(string); ok && prefix != "" {
		parts = append(parts, "path="+prefix)
	}
	if glob, ok := p.Criteria["nameGlob"].(string); ok && glob != "" {
		parts = append(parts, "name="+glob)
	}
	return strings.Join(parts, " ")
}

// cleanupIntCriteria reads a numeric criterion. The server stores the criteria
// as a free-form map, so a value that is not a number is not one.
func cleanupIntCriteria(criteria map[string]any, key string) (int, bool) {
	raw, ok := criteria[key]
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	default:
		return 0, false
	}
}

// cleanupPolicyByNameOrID finds the policy a command was given. The endpoints
// address a policy by its id, so a name has to be resolved to one.
func cleanupPolicyByNameOrID(nameOrID string) (*client.CleanupPolicy, error) {
	policies, err := nxsClient.CleanupList()
	if err != nil {
		return nil, err
	}
	for i := range policies {
		if policies[i].ID == nameOrID || policies[i].Name == nameOrID {
			return &policies[i], nil
		}
	}
	return nil, fmt.Errorf("cleanup policy %q not found", nameOrID)
}

// cleanupPolicyOptions turns the flags of "cleanup create" and "cleanup
// update" into a policy. A flag that was not given leaves the setting it
// would change alone.
func cleanupPolicyOptions(cmd *cobra.Command, policy client.CleanupPolicy) (client.CleanupPolicy, bool, error) {
	flags := cmd.Flags()
	given := false

	if flags.Changed("format") {
		format, _ := flags.GetString("format")
		if format == "" {
			format = "*"
		}
		policy.Format = format
		given = true
	}
	if flags.Changed("description") {
		v, _ := flags.GetString("description")
		policy.Description = v
		given = true
	}
	if flags.Changed("cron") {
		v, _ := flags.GetString("cron")
		policy.ScheduleCron = v
		given = true
	}
	if flags.Changed("dry-run") {
		v, _ := flags.GetBool("dry-run")
		policy.DryRun = v
		given = true
	}
	if flags.Changed("enabled") {
		v, _ := flags.GetBool("enabled")
		policy.Enabled = v
		given = true
	}
	if flags.Changed("retain-versions") {
		v, _ := flags.GetInt("retain-versions")
		if v < 0 {
			return policy, given, fmt.Errorf("--retain-versions must not be negative")
		}
		policy.RetainNVersions = v
		given = true
	}
	if flags.Changed("repository") {
		v, _ := flags.GetString("repository")
		policy.Scope.RepositoryName = v
		given = true
	}
	if flags.Changed("path-prefix") {
		v, _ := flags.GetString("path-prefix")
		policy.Scope.PathPrefix = v
		given = true
	}
	if policy.Criteria == nil {
		policy.Criteria = map[string]any{}
	}
	if flags.Changed("age") {
		v, _ := flags.GetInt("age")
		if v < 0 {
			return policy, given, fmt.Errorf("--age must not be negative")
		}
		policy.Criteria["artifactAgeDays"] = v
		given = true
	}
	if flags.Changed("last-downloaded-days") {
		v, _ := flags.GetInt("last-downloaded-days")
		if v < 0 {
			return policy, given, fmt.Errorf("--last-downloaded-days must not be negative")
		}
		policy.Criteria["lastDownloadedDays"] = v
		given = true
	}
	if flags.Changed("name-glob") {
		v, _ := flags.GetString("name-glob")
		policy.Criteria["nameGlob"] = v
		given = true
	}
	return policy, given, nil
}

// cleanupPolicyFlags adds the flags that describe a policy to a command.
func cleanupPolicyFlags(cmd *cobra.Command) {
	cmd.Flags().String("format", "", "Repository format the policy applies to (* for every format)")
	cmd.Flags().String("description", "", "Description of the policy")
	cmd.Flags().String("cron", "", "Cron schedule the policy runs on")
	cmd.Flags().Bool("dry-run", false, "Report what the policy would remove without removing it")
	cmd.Flags().Bool("enabled", false, "Let the policy run on its schedule")
	cmd.Flags().Int("retain-versions", 0, "Number of versions to keep per path")
	cmd.Flags().String("repository", "", "Repository the policy applies to")
	cmd.Flags().String("path-prefix", "", "Path prefix the policy applies to")
	cmd.Flags().Int("age", 0, "Remove artifacts older than this many days")
	cmd.Flags().Int("last-downloaded-days", 0, "Remove artifacts not downloaded for this many days")
	cmd.Flags().String("name-glob", "", "Glob the artifact names must match")
}

var cleanupCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a cleanup policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		policy := client.CleanupPolicy{
			Name:     args[0],
			Format:   "*",
			Criteria: map[string]any{},
		}
		policy, _, err := cleanupPolicyOptions(cmd, policy)
		if err != nil {
			return err
		}
		if err := nxsClient.CleanupCreate(policy); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Cleanup policy %q created", args[0]))
		return nil
	},
}

var cleanupUpdateCmd = &cobra.Command{
	Use:   "update <name|id>",
	Short: "Update a cleanup policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		existing, err := cleanupPolicyByNameOrID(args[0])
		if err != nil {
			return err
		}
		policy, given, err := cleanupPolicyOptions(cmd, *existing)
		if err != nil {
			return err
		}
		if !given {
			return fmt.Errorf("no update given — pass at least one of --format, --description, --cron, --dry-run, --enabled, --retain-versions, --repository, --path-prefix, --age, --last-downloaded-days, --name-glob")
		}
		if err := nxsClient.CleanupUpdate(existing.ID, policy); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Cleanup policy %q updated", existing.Name))
		return nil
	},
}

var cleanupDeleteCmd = &cobra.Command{
	Use:   "delete <name|id>",
	Short: "Delete a cleanup policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		existing, err := cleanupPolicyByNameOrID(args[0])
		if err != nil {
			return err
		}
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Delete cleanup policy %q? This cannot be undone. [y/N]: ", existing.Name)
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}
		if err := nxsClient.CleanupDelete(existing.ID); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Cleanup policy %q deleted", existing.Name))
		return nil
	},
}

var cleanupPreviewCmd = &cobra.Command{
	Use:   "preview <name|id>",
	Short: "Show what a cleanup policy would remove, without removing it",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		existing, err := cleanupPolicyByNameOrID(args[0])
		if err != nil {
			return err
		}
		preview, err := nxsClient.CleanupPreview(existing.ID)
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(preview)
			return nil
		}
		rows := make([][]string, 0, len(preview.Assets))
		for _, a := range preview.Assets {
			lastDownloaded := "never"
			if a.LastDownloaded != nil {
				lastDownloaded = a.LastDownloaded.Format("2006-01-02 15:04:05")
			}
			rows = append(rows, []string{
				a.Repository,
				a.Path,
				fmt.Sprintf("%d", a.SizeBytes),
				lastDownloaded,
				a.Reason,
			})
		}
		printer.Table([]string{"REPOSITORY", "PATH", "SIZE", "LAST DOWNLOADED", "REASON"}, rows)
		printer.Success(fmt.Sprintf("Policy %q would remove %d artifacts (%d bytes)", existing.Name, preview.TotalCount, preview.TotalBytes))
		return nil
	},
}

var cleanupRunCmd = &cobra.Command{
	Use:   "run <policy-name>",
	Short: "Execute a cleanup policy immediately",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		existing, err := cleanupPolicyByNameOrID(args[0])
		if err != nil {
			return err
		}
		if err := nxsClient.CleanupRun(existing.ID); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Cleanup policy %q triggered", existing.Name))
		return nil
	},
}

func init() {
	cleanupPolicyFlags(cleanupCreateCmd)
	cleanupPolicyFlags(cleanupUpdateCmd)
	cleanupDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	cleanupCmd.AddCommand(cleanupListCmd, cleanupCreateCmd, cleanupUpdateCmd, cleanupDeleteCmd, cleanupPreviewCmd, cleanupRunCmd)
	rootCmd.AddCommand(cleanupCmd)
}
