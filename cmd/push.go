package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/nexspence/nxs/internal/output"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <repo> <remote-path> <local-file>",
	Short: "Upload an artifact to a repository",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		repo, remotePath, localFile := args[0], args[1], args[2]

		progressFn := func(size int64) io.Writer {
			bar := output.NewProgress(size, "Uploading "+filepath.Base(localFile), flagJSON, flagPlain)
			if bar == nil {
				return nil
			}
			return bar
		}

		if err := nxsClient.Push(repo, remotePath, localFile, progressFn); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Uploaded %s → %s/%s", localFile, repo, remotePath))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
