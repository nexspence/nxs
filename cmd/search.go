package cmd

import (
	"github.com/nexspence/nxs/internal/client"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for components across repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		repo, _ := cmd.Flags().GetString("repo")
		format, _ := cmd.Flags().GetString("format")
		query, _ := cmd.Flags().GetString("q")
		tag, _ := cmd.Flags().GetString("tag")

		results, err := nxsClient.Search(client.SearchParams{
			Repo:   repo,
			Format: format,
			Query:  query,
			Tag:    tag,
		})
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(results)
			return nil
		}
		rows := make([][]string, 0, len(results))
		for _, r := range results {
			rows = append(rows, []string{r.Repository, r.Format, r.Group, r.Name, r.Version})
		}
		printer.Table([]string{"REPO", "FORMAT", "GROUP", "NAME", "VERSION"}, rows)
		return nil
	},
}

func init() {
	searchCmd.Flags().String("repo", "", "Filter by repository name")
	searchCmd.Flags().String("format", "", "Filter by format")
	searchCmd.Flags().StringP("q", "q", "", "Keyword search")
	searchCmd.Flags().String("tag", "", "Filter by tag (key=value)")
	rootCmd.AddCommand(searchCmd)
}
