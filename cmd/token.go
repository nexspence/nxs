package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage API tokens",
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your API tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		toks, err := nxsClient.TokenList()
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(toks)
			return nil
		}
		rows := make([][]string, 0, len(toks))
		for _, t := range toks {
			expires := "never"
			if t.ExpiresAt != nil {
				expires = t.ExpiresAt.Format(time.RFC3339)
			}
			last := "never"
			if t.LastUsed != nil {
				last = t.LastUsed.Format(time.RFC3339)
			}
			rows = append(rows, []string{t.ID, t.Name, strings.Join(t.Scopes, ","), expires, last})
		}
		printer.Table([]string{"ID", "NAME", "SCOPES", "EXPIRES", "LAST USED"}, rows)
		return nil
	},
}

var tokenCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new API token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		scopes, _ := cmd.Flags().GetStringSlice("scope")
		days, _ := cmd.Flags().GetInt("expires-days")
		var expPtr *int
		if days > 0 {
			expPtr = &days
		}
		tok, err := nxsClient.TokenCreate(args[0], scopes, expPtr)
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(tok)
			return nil
		}
		printer.Success(fmt.Sprintf("Token %q created", tok.Name))
		fmt.Println(tok.Token)
		fmt.Fprintln(cmd.ErrOrStderr(), "Save this token now — it will not be shown again.")
		return nil
	},
}

var tokenDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an API token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		if err := nxsClient.TokenDelete(args[0]); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Token %s deleted", args[0]))
		return nil
	},
}

func init() {
	tokenCreateCmd.Flags().StringSlice("scope", nil, "Scope to grant (repeatable)")
	tokenCreateCmd.Flags().Int("expires-days", 0, "Days until expiry (0 = server default / never)")
	tokenCmd.AddCommand(tokenListCmd, tokenCreateCmd, tokenDeleteCmd)
	rootCmd.AddCommand(tokenCmd)
}
