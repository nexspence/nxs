package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var blobstoreCmd = &cobra.Command{
	Use:   "blobstore",
	Short: "Manage blob stores",
}

var blobstoreCompactCmd = &cobra.Command{
	Use:   "compact <name>",
	Short: "Run garbage collection on a blob store (remove unreferenced blobs)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		minAge, _ := cmd.Flags().GetString("min-age")
		res, err := nxsClient.BlobStoreCompact(args[0], dryRun, minAge)
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(res)
			return nil
		}
		verb := "Collected"
		if res.DryRun {
			verb = "Would collect"
		}
		printer.Success(fmt.Sprintf("%s %d orphan(s), %d bytes freed (%d blobs scanned) in %q",
			verb, res.Orphans, res.FreedBytes, res.ScannedBlobs, res.Store))
		return nil
	},
}

func init() {
	blobstoreCompactCmd.Flags().Bool("dry-run", false, "report orphans without deleting them")
	blobstoreCompactCmd.Flags().String("min-age", "", "only collect orphans older than this (e.g. 24h); overrides server default")
	blobstoreCmd.AddCommand(blobstoreCompactCmd)
	rootCmd.AddCommand(blobstoreCmd)
}
