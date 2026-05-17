package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/nexspence/nxs/internal/output"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <repo> <remote-path>",
	Short: "Download an artifact from a repository",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		repo, remotePath := args[0], args[1]
		outDir, _ := cmd.Flags().GetString("output")

		filename := filepath.Base(remotePath)
		localPath := filepath.Join(outDir, filename)

		progressFn := func(size int64) io.Writer {
			bar := output.NewProgress(size, "Downloading "+filename, flagJSON, flagPlain)
			if bar == nil {
				return nil
			}
			return bar
		}

		if err := nxsClient.Pull(repo, remotePath, localPath, progressFn); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Downloaded %s/%s → %s", repo, remotePath, localPath))
		return nil
	},
}

func init() {
	pullCmd.Flags().StringP("output", "o", ".", "Output directory")
	rootCmd.AddCommand(pullCmd)
}
