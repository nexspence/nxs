package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/nexspence/nxs/internal/batch"
	"github.com/nexspence/nxs/internal/output"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <repo> <remote-prefix> <local>",
	Short: "Upload an artifact (or a directory/glob with -r) to a repository",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		repo, remotePrefix, local := args[0], args[1], args[2]
		recursive, _ := cmd.Flags().GetBool("recursive")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

		jobs, err := batch.Walk(local, recursive)
		if err != nil {
			return err
		}

		// Single file → preserve the original per-file progress bar UX.
		if len(jobs) == 1 && !recursive && jobs[0].RelPath == filepath.Base(local) {
			remotePath := joinRemote(remotePrefix, jobs[0].RelPath)
			progressFn := func(size int64) io.Writer {
				bar := output.NewProgress(size, "Uploading "+filepath.Base(local), flagJSON, flagPlain)
				if bar == nil {
					return nil
				}
				return bar
			}
			if err := nxsClient.Push(repo, remotePath, jobs[0].LocalPath, progressFn); err != nil {
				return err
			}
			printer.Success(fmt.Sprintf("Uploaded %s → %s/%s", local, repo, remotePath))
			return nil
		}

		total := len(jobs)
		done := 0
		res := batch.RunPool(jobs, concurrency, continueOnError, func(j batch.Job) error {
			remotePath := joinRemote(remotePrefix, j.RelPath)
			err := nxsClient.Push(repo, remotePath, j.LocalPath, nil)
			done++
			if !flagJSON {
				fmt.Fprintf(cmd.ErrOrStderr(), "[%d/%d] %s\n", done, total, j.RelPath)
			}
			return err
		})

		printer.Success(fmt.Sprintf("%d uploaded, %d failed", res.OK, len(res.Failed)))
		for _, e := range res.Failed {
			printer.Error(e.Error())
		}
		if len(res.Failed) > 0 {
			return fmt.Errorf("%d uploads failed", len(res.Failed))
		}
		return nil
	},
}

// joinRemote joins a remote prefix and a relative path with a single slash,
// tolerating empty/"." prefixes.
func joinRemote(prefix, rel string) string {
	if prefix == "" || prefix == "." || prefix == "/" {
		return rel
	}
	return prefix + "/" + rel
}

func init() {
	pushCmd.Flags().BoolP("recursive", "r", false, "Upload a directory tree")
	pushCmd.Flags().Int("concurrency", 4, "Parallel uploads")
	pushCmd.Flags().Bool("continue-on-error", false, "Keep going after a failed upload")
	rootCmd.AddCommand(pushCmd)
}
