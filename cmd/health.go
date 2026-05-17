package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show server health and version",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		watch, _ := cmd.Flags().GetBool("watch")
		for {
			info, err := nxsClient.SystemInfo()
			if err != nil {
				printer.Error(err.Error())
			} else if flagJSON {
				printer.JSON(info)
			} else {
				printer.Table([]string{"FIELD", "VALUE"}, [][]string{
					{"Status", info.Status},
					{"Version", info.Version},
					{"App", info.AppName},
				})
			}
			if !watch {
				break
			}
			fmt.Println("\n(refreshing every 5s — Ctrl+C to stop)")
			time.Sleep(5 * time.Second)
		}
		return nil
	},
}

func init() {
	healthCmd.Flags().Bool("watch", false, "Refresh every 5 seconds")
	rootCmd.AddCommand(healthCmd)
}
