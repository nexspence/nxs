// Package batch expands local file selections into upload jobs and runs work
// concurrently with a bounded worker pool.
package batch

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
)

// Job is a single file to transfer. RelPath is the path used to build the
// remote location (always forward-slash separated).
type Job struct {
	LocalPath string
	RelPath   string
}

// Result aggregates a pool run.
type Result struct {
	OK     int
	Failed []error
}

func hasGlob(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

// Walk turns local (a file, a directory, or a glob pattern) into jobs.
//   - single file: one job, RelPath = base name.
//   - directory (recursive=true): every file underneath, RelPath relative to dir.
//   - glob: every match, RelPath relative to the longest non-glob base segment.
func Walk(local string, recursive bool) ([]Job, error) {
	if hasGlob(local) {
		return walkGlob(local)
	}
	info, err := os.Stat(local)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		if !recursive {
			return nil, fmt.Errorf("%s is a directory; pass -r to upload recursively", local)
		}
		return walkDir(local)
	}
	return []Job{{LocalPath: local, RelPath: filepath.Base(local)}}, nil
}

func walkDir(dir string) ([]Job, error) {
	var jobs []Job
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		jobs = append(jobs, Job{LocalPath: path, RelPath: filepath.ToSlash(rel)})
		return nil
	})
	return jobs, err
}

func walkGlob(pattern string) ([]Job, error) {
	base, _ := doublestar.SplitPattern(filepath.ToSlash(pattern))
	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, err
	}
	var jobs []Job
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil || info.IsDir() {
			continue
		}
		rel := filepath.Base(m)
		if base != "" && base != "." {
			if r, err := filepath.Rel(base, m); err == nil {
				rel = r
			}
		}
		jobs = append(jobs, Job{LocalPath: m, RelPath: filepath.ToSlash(rel)})
	}
	return jobs, nil
}

// RunPool executes fn over jobs with up to concurrency workers. When
// continueOnError is false the first error stops new work from starting.
func RunPool(jobs []Job, concurrency int, continueOnError bool, fn func(Job) error) Result {
	if concurrency < 1 {
		concurrency = 1
	}
	var (
		mu      sync.Mutex
		res     Result
		stopped bool
		wg      sync.WaitGroup
	)
	sem := make(chan struct{}, concurrency)
	for _, j := range jobs {
		mu.Lock()
		if stopped {
			mu.Unlock()
			break
		}
		mu.Unlock()

		wg.Add(1)
		sem <- struct{}{}
		go func(j Job) {
			defer wg.Done()
			defer func() { <-sem }()
			err := fn(j)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				res.Failed = append(res.Failed, fmt.Errorf("%s: %w", j.RelPath, err))
				if !continueOnError {
					stopped = true
				}
			} else {
				res.OK++
			}
		}(j)
	}
	wg.Wait()
	return res
}
