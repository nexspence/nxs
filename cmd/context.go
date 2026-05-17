package cmd

import (
	"fmt"

	"github.com/nexspence/nxs/internal/config"
	"github.com/nexspence/nxs/internal/output"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage named server contexts",
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured contexts",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("")
		if err != nil {
			return err
		}
		p := output.NewPrinter(flagJSON, flagPlain)
		names := cfg.ListContexts()
		rows := make([][]string, 0, len(names))
		for _, n := range names {
			marker := ""
			if n == cfg.ActiveContext() {
				marker = "*"
			}
			rows = append(rows, []string{marker, n})
		}
		p.Table([]string{"", "NAME"}, rows)
		return nil
	},
}

var contextUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch the active context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("")
		if err != nil {
			return err
		}
		name := args[0]
		ctx, ok := cfg.Contexts[name]
		if !ok {
			return fmt.Errorf("context %q not found", name)
		}
		if err := cfg.Save("", name, ctx.URL, ctx.Token); err != nil {
			return err
		}
		fmt.Printf("Switched to context %q\n", name)
		return nil
	},
}

func init() {
	contextCmd.AddCommand(contextListCmd, contextUseCmd)
	rootCmd.AddCommand(contextCmd)
}
