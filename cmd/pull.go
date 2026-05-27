package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/nexspence/nxs/internal/batch"
	"github.com/nexspence/nxs/internal/output"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <repo> <remote-path-or-prefix>",
	Short: "Download an artifact (or a path prefix with -r) from a repository",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		repo, remote := args[0], args[1]
		outDir, _ := cmd.Flags().GetString("output")
		recursive, _ := cmd.Flags().GetBool("recursive")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

		if !recursive {
			filename := filepath.Base(remote)
			localPath := filepath.Join(outDir, filename)
			progressFn := func(size int64) io.Writer {
				bar := output.NewProgress(size, "Downloading "+filename, flagJSON, flagPlain)
				if bar == nil {
					return nil
				}
				return bar
			}
			if err := nxsClient.Pull(repo, remote, localPath, progressFn); err != nil {
				return err
			}
			printer.Success(fmt.Sprintf("Downloaded %s/%s → %s", repo, remote, localPath))
			return nil
		}

		assets, err := nxsClient.SearchAssets(repo, remote)
		if err != nil {
			return err
		}
		if len(assets) == 0 {
			return fmt.Errorf("no assets under %q in repo %q", remote, repo)
		}

		jobs := make([]batch.Job, 0, len(assets))
		for _, a := range assets {
			jobs = append(jobs, batch.Job{LocalPath: a.Path, RelPath: a.Path})
		}

		total := len(jobs)
		var done atomic.Int64
		res := batch.RunPool(jobs, concurrency, continueOnError, func(j batch.Job) error {
			localPath := filepath.Join(outDir, filepath.FromSlash(j.RelPath))
			if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
				return err
			}
			err := nxsClient.Pull(repo, j.RelPath, localPath, nil)
			n := done.Add(1)
			if !flagJSON {
				fmt.Fprintf(cmd.ErrOrStderr(), "[%d/%d] %s\n", n, total, j.RelPath)
			}
			return err
		})

		printer.Success(fmt.Sprintf("%d downloaded, %d failed", res.OK, len(res.Failed)))
		for _, e := range res.Failed {
			printer.Error(e.Error())
		}
		if len(res.Failed) > 0 {
			return fmt.Errorf("%d downloads failed", len(res.Failed))
		}
		return nil
	},
}

func init() {
	pullCmd.Flags().StringP("output", "o", ".", "Output directory")
	pullCmd.Flags().BoolP("recursive", "r", false, "Download every asset under the path prefix")
	pullCmd.Flags().Int("concurrency", 4, "Parallel downloads")
	pullCmd.Flags().Bool("continue-on-error", false, "Keep going after a failed download")
	rootCmd.AddCommand(pullCmd)
}
