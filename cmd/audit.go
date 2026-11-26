package cmd

import (
	"fmt"

	"github.com/nexspence/nxs/internal/client"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Read the audit log",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		flags := cmd.Flags()
		limit, _ := flags.GetInt("limit")
		offset, _ := flags.GetInt("offset")
		domain, _ := flags.GetString("domain")
		action, _ := flags.GetString("action")
		username, _ := flags.GetString("username")
		from, _ := flags.GetString("from")
		to, _ := flags.GetString("to")

		entries, err := nxsClient.AuditList(client.AuditParams{
			Limit:    limit,
			Offset:   offset,
			Domain:   domain,
			Action:   action,
			Username: username,
			From:     from,
			To:       to,
		})
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(entries)
			return nil
		}
		rows := make([][]string, 0, len(entries))
		for _, e := range entries {
			entity := e.EntityName
			if entity == "" {
				entity = e.EntityID
			}
			rows = append(rows, []string{
				e.EventTime.Format("2006-01-02 15:04:05"),
				e.Username,
				e.Domain,
				e.Action,
				entity,
				e.Result,
			})
		}
		printer.Table([]string{"TIME", "USER", "DOMAIN", "ACTION", "ENTITY", "RESULT"}, rows)
		if len(entries) == 0 {
			fmt.Println("No audit entries.")
		}
		return nil
	},
}

func init() {
	auditCmd.Flags().Int("limit", 50, "Maximum number of entries to read")
	auditCmd.Flags().Int("offset", 0, "Number of entries to skip")
	auditCmd.Flags().String("domain", "", "Filter by domain (REPOSITORY, SECURITY, USER, ...)")
	auditCmd.Flags().String("action", "", "Filter by action (CREATE, DELETE, LOGIN, ...)")
	auditCmd.Flags().String("username", "", "Filter by user name")
	auditCmd.Flags().String("from", "", "Only entries from this date (YYYY-MM-DD or RFC3339)")
	auditCmd.Flags().String("to", "", "Only entries up to this date (YYYY-MM-DD or RFC3339)")

	rootCmd.AddCommand(auditCmd)
}
